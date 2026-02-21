package parser

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"ts_inspector/ast"
	"ts_inspector/ast/indexing"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var logger = utils.GetLogger("parser_state")

type fileState struct {
	Classes            []*Class
	Content            string
	DynamicImportFiles []*File
	DynamicImportPaths []string
	Exports            References
	Filetype           string
	Imports            []*ast.ImportParseResult
	IsOpen             bool
	LineOffsets        []uint32
	URI                string
	Variables          []*Variable
	Version            int
}

type File struct {
	sync.RWMutex
	state fileState
}

func (f *File) Filename() string { return FilenameFromUri(f.Snapshot().URI) }

func (f *File) FindImportPath(identifier string) string {
	for _, importClause := range f.Snapshot().Imports {
		for _, imp := range importClause.Imports {
			if imp.LocalIdentifier == identifier {
				return importClause.Package
			}
		}
	}

	return ""
}

func (f *File) GetDependencies(state *State) []string {
	dependents := make([]string, 0)

	for _, class := range *state.GetClasses() {
		if class.GetTemplateFile() == f {
			dependents = append(dependents, class.Snapshot().File.Filename())
		}
	}

	for _, class := range *state.GetClasses() {
		for _, fileClass := range f.Snapshot().Classes {
			if class.DoesExtendOrImplement(fileClass) {
				dependents = append(dependents, class.Snapshot().File.Filename())
			}
		}

		if !class.HasModule() {
			continue
		}

		for d := range class.Snapshot().Angular.Module.Declarations.FlattenReferenceArraysToReferences(state) {
			d.Resolve(state)
			if d.Class == nil {
				continue
			}

			if d.Class.Snapshot().File == f || d.Class.GetTemplateFile() == f {
				dependents = append(dependents, class.Snapshot().File.Filename())

				eih := d.Class.GetExtendsImplementsHierarchy()
				for _, c := range eih {
					dependents = append(dependents, c.Snapshot().File.Filename())
				}
			}

			for _, dep := range dependents {
				if d.Class.Snapshot().File.Filename() == dep {
					dependents = append(dependents, d.Class.Snapshot().File.Filename())
				}
			}
		}
	}

	for _, class := range f.Snapshot().Classes {
		eih := class.GetExtendsImplementsHierarchy()
		for _, c := range eih {
			dependents = append(dependents, c.Snapshot().File.Filename())
		}

		t := class.GetTemplateFile()

		if t == nil {
			continue
		}

		if t == f {
			dependents = append(dependents, t.Filename())
		}

		if class.Snapshot().File.Filename() != t.Filename() {
			dependents = append(dependents, t.Filename())
		}

		// If there's a template, class.Angular.Component cannot not be nil
		for _, module := range class.Snapshot().Angular.Component.DeclaredIn {
			dependents = append(dependents, module.Snapshot().File.Filename())
		}
	}

	slices.Sort(dependents)

	return slices.Compact(dependents)
}

func (f *File) GetInterestingPoints() []InterestingPoint {
	interestingPoints := make([]InterestingPoint, 0)
	for _, class := range f.Snapshot().Classes {
		interestingPoints = append(interestingPoints, class.GetInterestingPoints()...)
	}

	return interestingPoints
}

func (f *File) GetOffsetForPosition(p utils.Position) uint32 {
	file := f.Snapshot()
	lines := uint32(len(file.LineOffsets))

	if p.Line >= lines {
		return uint32(len(file.Content))
	} else if p.Line < 0 {
		return 0
	}

	lineOffset := file.LineOffsets[p.Line]

	var nextLineOffset uint32

	if p.Line+1 < lines {
		nextLineOffset = file.LineOffsets[p.Line+1]
	} else {
		nextLineOffset = lines
	}

	return max(min(lineOffset+p.Character, nextLineOffset), lineOffset)
}

func (f *File) GetOffsetsForRange(r utils.Range) (uint32, uint32) {
	return f.GetOffsetForPosition(r.Start), f.GetOffsetForPosition(r.End)
}

func (f *File) Postprocess(state *State) {
	file := f.Snapshot()

	for _, class := range file.Classes {
		state.SetClass(class.Id(), class)
		class.Postprocess(state)
	}

	f.ResolveDynamicallyImportedFiles(state)

	if file.Filetype == "pug" {
		for _, class := range *state.GetClasses() {
			snapshot := class.Snapshot()
			if snapshot.Angular != nil &&
				snapshot.Angular.Component != nil &&
				snapshot.Angular.Component.TemplateUrlFile != nil &&
				snapshot.Angular.Component.TemplateUrlFile.Snapshot().URI == file.URI {

				f.Update(func(data *fileState) {
					data.Classes = append(data.Classes, class)
				})
			}
		}
	}
}

func (f *File) ResolveDynamicallyImportedFiles(state *State) {
	file := f.Snapshot()

	dynamicImportFiles := make([]*File, len(file.DynamicImportPaths))

	var wg sync.WaitGroup

	for i, importPath := range file.DynamicImportPaths {
		wg.Go(func() {
			absolutePath, err := filepath.Abs(path.Join(filepath.Dir(FilenameFromUri(file.URI)), importPath))
			if err != nil {
				logger.Println(err)
				return
			}

			resolvedFile := getFileByPath(state, absolutePath)
			if resolvedFile == nil {
				return
			}

			dynamicImportFiles[i] = resolvedFile
		})

		if !utils.Concurrency {
			wg.Wait()
		}
	}

	wg.Wait()

	f.Update(func(data *fileState) {
		data.DynamicImportFiles = dynamicImportFiles
	})
}

func (f *File) ResetDeclarations() {
	f.Update(func(data *fileState) {
		data.Classes = make([]*Class, 0)
		data.Exports = make(References, 0)
		data.Variables = make([]*Variable, 0)
	})
}

func (f *File) SetContent(content string, version int) {
	lineOffsets := utils.GetLineOffsets(content)

	f.Update(func(data *fileState) {
		data.LineOffsets = lineOffsets
		data.Content = content
		data.Version = versionFallback(version, data.URI)
	})
}

func (f *File) SetOpen() {
	f.Update(func(data *fileState) {
		data.IsOpen = true
	})
}

func (f *File) Snapshot() fileState {
	f.RLock()
	state := f.state
	f.RUnlock()

	return state
}

func (f *File) Update(fn func(data *fileState)) {
	f.Lock()
	defer f.Unlock()
	fn(&f.state)
}

func FiletypeFromFilename(filename string) (string, error) {
	if strings.HasSuffix(filename, ".pug") {
		return "pug", nil
	} else if strings.HasSuffix(filename, ".ts") {
		return "typescript", nil
	} else if strings.HasSuffix(filename, ".html") {
		return "html", nil
	}

	return "", fmt.Errorf("Couldn't determine filetype from filename: %s", filename)
}

func IndexFileFromIndexer(state *State, filename string) error {
	var err error

	filetype, err := FiletypeFromFilename(filename)
	if err != nil {
		return nil // unhandled file type, not really an error
	}

	switch filetype {
	case "typescript":
		err = IndexTypeScriptFileFromIndexer(state, filename)
	case "pug":
		err = IndexPugFromIndexer(state, filename)
	}

	if err != nil {
		return err
	}

	file, found := state.GetFile(filename)
	if found {
		file.Postprocess(state)
	}

	return nil
}

func IndexFileFromLsp(state *State, uri string, languageId string, version int, content string, logger *log.Logger) error {
	var err error

	language := languageId
	if language == "" {
		filetype, err := FiletypeFromFilename(FilenameFromUri(uri))

		if err != nil {
			return nil // unhandled file type, not really an error
		}

		language = filetype
	}
	switch language {
	case "typescript":
		err = IndexTypeScriptFileFromLsp(state, uri, languageId, version, content, logger)
	case "pug":
		err = IndexPugFileFromLsp(state, uri, content, version)
	}

	return err
}

func NewFile(uri string, filetype string, version int) (*File, error) {
	filename := FilenameFromUri(uri)

	if !strings.HasPrefix(filename, "/") {
		cwd, err := os.Getwd()

		if err != nil {
			file := File{}

			return &file, err
		}

		filename, err = filepath.Abs(path.Join(cwd, filename))
		if err != nil {
			file := File{}

			return &file, err
		}
	}

	fileState := fileState{Classes: []*Class{}, Content: "", DynamicImportFiles: []*File{}, DynamicImportPaths: []string{}, Exports: []*Reference{}, Filetype: filetype, Imports: []*ast.ImportParseResult{}, LineOffsets: []uint32{}, URI: UriFromFilename(filename), Version: version}
	file := File{state: fileState}

	return &file, nil
}

func createFileIfNotExists(state *State, filename string, content string, version int) (*File, error) {
	file, found := state.GetFile(filename)

	if !found {
		uri := UriFromFilename(filename)
		filetype, err := FiletypeFromFilename(filename)

		if err != nil {
			return nil, err
		}

		file, err = NewFile(uri, filetype, versionFallback(0, uri))
		if err != nil {
			return nil, err
		}

		if content != "" {
			file.SetContent(content, version)
		} else {
			_, err = utils.ParseFile(true, file.Filename(), filetype, nil, func(root *sitter.Node, content []byte, _ any) (any, error) {
				file.SetContent(CStr2GoStr(content), version)
				return nil, nil
			})
		}

		state.SetFile(file.Filename(), file)
	} else {
		if content != "" || version != 0 {
			file.SetContent(content, version)
		}
	}

	return file, nil
}

func getFileByPath(state *State, path string) *File {
	file, found := state.GetFile(path)
	if found {
		return file
	}

	extensionedPath, found := indexing.DetermineFilename(path)
	if !found {
		return nil
	}

	file, found = state.GetFile(extensionedPath)
	if found {
		return file
	}

	IndexFileFromIndexer(state, extensionedPath)
	file, found = state.GetFile(extensionedPath)
	if found {
		return file
	}

	return nil
}

func resolveProjectImportPath(state *State, currentFile *File, importPath string) *File {
	absolutePath, err := filepath.Abs(path.Join(filepath.Dir(FilenameFromUri(currentFile.Snapshot().URI)), importPath))

	if err != nil {
		logger.Println(err)
		return nil
	}

	resolvedFile := getFileByPath(state, absolutePath)
	if resolvedFile != nil {
		return resolvedFile
	}

	return nil
}

func versionFallback(version int, uri string) int {
	v := version

	if v == 0 {
		lastSeenVersion, found := lastSeenDocumentVersion[uri]
		if found {
			v = lastSeenVersion
		}
	} else {
		lastSeenDocumentVersion[uri] = v
	}

	return v
}

var lastSeenDocumentVersion = make(map[string]int, 0)
