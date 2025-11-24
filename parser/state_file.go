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
	LineOffsets        []uint32
	URI                string
	Version            int
}

type File struct {
	sync.RWMutex
	state fileState
}

func (c *File) Filename() string { return FilenameFromUri(c.Snapshot().URI) }

func (c *File) FindImportPath(identifier string) string {
	for _, importClause := range c.Snapshot().Imports {
		for _, imp := range importClause.Imports {
			if imp.LocalIdentifier == identifier {
				return importClause.Package
			}
		}
	}

	return ""
}

func (c *File) GetDependencies(state *State) []string {
	dependents := make([]string, 0)

	for _, class := range *state.GetClasses() {
		if class.GetTemplateFile() == c {
			dependents = append(dependents, class.Snapshot().File.Filename())
		}
	}

	for _, class := range *state.GetClasses() {
		if !class.HasModule() {
			continue
		}

		for d := range class.Snapshot().Angular.Module.Declarations.IterateResolved {
			if d.Class.Snapshot().File == c || d.Class.GetTemplateFile() == c {
				dependents = append(dependents, class.Snapshot().File.Filename())
			}

			for _, dep := range dependents {
				if d.Class.Snapshot().File.Filename() == dep {
					dependents = append(dependents, d.Class.Snapshot().File.Filename())
				}
			}
		}
	}

	for _, class := range c.Snapshot().Classes {
		t := class.GetTemplateFile()

		if t == nil {
			continue
		}

		if t == c {
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

func (c *File) GetOffsetForPosition(p utils.Position) uint32 {
	file := c.Snapshot()
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

func (c *File) GetOffsetsForRange(r utils.Range) (uint32, uint32) {
	return c.GetOffsetForPosition(r.Start), c.GetOffsetForPosition(r.End)
}

func (c *File) Postprocess(state *State) {
	file := c.Snapshot()

	for _, class := range file.Classes {
		state.SetClass(class.Id(), class)
		class.Postprocess(state)
	}

	c.ResolveDynamicallyImportedFiles(state)

	if file.Filetype == "pug" {
		for _, class := range *state.GetClasses() {
			snapshot := class.Snapshot()
			if snapshot.Angular != nil &&
				snapshot.Angular.Component != nil &&
				snapshot.Angular.Component.TemplateUrlFile != nil &&
				snapshot.Angular.Component.TemplateUrlFile.Snapshot().URI == file.URI {

				c.Update(func(data *fileState) {
					data.Classes = append(data.Classes, class)
				})
			}
		}
	}
}

func (c *File) ResolveDynamicallyImportedFiles(state *State) {
	file := c.Snapshot()

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

	c.Update(func(data *fileState) {
		data.DynamicImportFiles = dynamicImportFiles
	})
}

func (c *File) ResetClasses() {
	c.Update(func(data *fileState) {
		data.Classes = make([]*Class, 0)
		data.Exports = make(References, 0)
	})
}

func (c *File) SetContent(content string, version int) {
	lineOffsets := getLineOffsets(content)

	c.Update(func(data *fileState) {
		data.LineOffsets = lineOffsets
		data.Content = content
		data.Version = versionFallback(version, data.URI)
	})
}

func (c *File) Snapshot() fileState {
	c.RLock()
	state := c.state
	c.RUnlock()

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

func GetPositionForOffset(content string, offset uint32) utils.Position {
	lineOffsets := getLineOffsets(content)

	if offset >= uint32(len(content)) {
		return utils.Position{Line: uint32(len(lineOffsets)), Character: 0}
	} else if offset < 0 {
		return utils.Position{Line: 0, Character: 0}
	}

	var line uint32
	var character uint32

	for index, lineOffset := range lineOffsets {
		if lineOffset > offset {
			if index > 0 {
				line = uint32(index - 1)
				character = offset - lineOffsets[index-1]
			} else {
				line = 0
				character = offset
			}

			break
		}
	}

	return utils.Position{Line: line, Character: character}
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

func getLineOffsets(text string) []uint32 {
	var i uint32 = 0

	offsets := []uint32{}
	isLineStart := true

	textLength := uint32(len(text))
	for i < textLength {
		if isLineStart {
			offsets = append(offsets, i)
			isLineStart = false
		}

		ch := text[i]
		isLineStart = ch == '\r' || ch == '\n'

		if ch == '\r' && i+1 < textLength && text[i+1] == '\n' {
			i++
		}

		i++
	}

	if isLineStart && textLength > 0 {
		offsets = append(offsets, textLength)
	}

	return offsets
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
