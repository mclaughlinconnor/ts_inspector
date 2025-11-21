package parser

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/ast/indexing"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var logger = utils.GetLogger("parser_state")

type File struct {
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

func (f *File) FindImportPath(identifier string) string {
	for _, importClause := range f.Imports {
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

	for _, class := range state.Classes {
		if class.GetTemplateFile() == f {
			dependents = append(dependents, class.File.Filename())
		}
	}

	for _, class := range state.Classes {
		if !class.HasModule() {
			continue
		}

		for d := range class.Angular.Module.Declarations.IterateResolved {
			if d.Class.File == f || d.Class.GetTemplateFile() == f {
				dependents = append(dependents, class.File.Filename())
			}

			for _, dep := range dependents {
				if d.Class.File.Filename() == dep {
					dependents = append(dependents, d.Class.File.Filename())
				}
			}
		}
	}

	for _, class := range f.Classes {
		t := class.GetTemplateFile()

		if t == nil {
			continue
		}

		if t == f {
			dependents = append(dependents, t.Filename())
		}

		if class.File.Filename() != t.Filename() {
			dependents = append(dependents, t.Filename())
		}

		// If there's a template, class.Angular.Component cannot not be nil
		for _, module := range class.Angular.Component.DeclaredIn {
			dependents = append(dependents, module.File.Filename())
		}
	}

	slices.Sort(dependents)

	return slices.Compact(dependents)
}

func (f *File) GetOffsetForPosition(p utils.Position) uint32 {
	lines := uint32(len(f.LineOffsets))

	if p.Line >= lines {
		return uint32(len(f.Content))
	} else if p.Line < 0 {
		return 0
	}

	lineOffset := f.LineOffsets[p.Line]

	var nextLineOffset uint32

	if p.Line+1 < lines {
		nextLineOffset = f.LineOffsets[p.Line+1]
	} else {
		nextLineOffset = lines
	}

	return max(min(lineOffset+p.Character, nextLineOffset), lineOffset)
}

func (f *File) GetOffsetsForRange(r utils.Range) (uint32, uint32) {
	return f.GetOffsetForPosition(r.Start), f.GetOffsetForPosition(r.End)
}

func (f *File) Postprocess(state *State) {
	for _, class := range f.Classes {
		state.Classes[class.Id()] = class
		class.Postprocess(state)
	}

	f.ResolveDynamicallyImportedFiles(state)

	if f.Filetype == "pug" {
		for _, class := range state.Classes {
			if class.Angular != nil &&
				class.Angular.Component != nil &&
				class.Angular.Component.TemplateUrlFile != nil &&
				class.Angular.Component.TemplateUrlFile.URI == f.URI {

				f.Classes = append(f.Classes, class)
			}
		}
	}
}

func (f *File) ResolveDynamicallyImportedFiles(state *State) {
	dynamicImportFiles := make([]*File, len(f.DynamicImportPaths))

	for i, importPath := range f.DynamicImportPaths {
		absolutePath, err := filepath.Abs(path.Join(filepath.Dir(FilenameFromUri(f.URI)), importPath))

		if err != nil {
			logger.Println(err)
			continue
		}

		resolvedFile := getFileByPath(state, absolutePath)
		if resolvedFile != nil {
			continue
		}

		dynamicImportFiles[i] = resolvedFile
	}

	f.DynamicImportFiles = dynamicImportFiles
}

func (f *File) ResetClasses() {
	f.Classes = make([]*Class, 0)
	f.Exports = make(References, 0)
}

func (f *File) SetContent(content string, version int) {
	lineOffsets := getLineOffsets(content)

	f.LineOffsets = lineOffsets
	f.Content = content
	f.Version = versionFallback(version, f.URI)
}

func (f File) Filename() string { return FilenameFromUri(f.URI) }

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

	file, found := state.Files[filename]
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

	file := File{Classes: []*Class{}, Content: "", DynamicImportFiles: []*File{}, DynamicImportPaths: []string{}, Exports: []*Reference{}, Filetype: filetype, Imports: []*ast.ImportParseResult{}, LineOffsets: []uint32{}, URI: UriFromFilename(filename), Version: version}

	return &file, nil
}

func createFileIfNotExists(state *State, filename string, content string, version int) (*File, error) {
	file, found := state.Files[filename]

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

		state.Files[file.Filename()] = file
	} else {
		if content != "" || version != 0 {
			file.SetContent(content, version)
		}
	}

	return file, nil
}

func getFileByPath(state *State, path string) *File {
	file, found := state.Files[path]
	if found {
		return file
	}

	extensionedPath, found := indexing.DetermineFilename(path)
	if !found {
		return nil
	}

	file, found = state.Files[extensionedPath]
	if found {
		return file
	}

	IndexFileFromIndexer(state, extensionedPath)
	file, found = state.Files[extensionedPath]
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
	absolutePath, err := filepath.Abs(path.Join(filepath.Dir(FilenameFromUri(currentFile.URI)), importPath))

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
