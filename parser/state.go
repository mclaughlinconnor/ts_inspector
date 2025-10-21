package parser

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type access struct {
	Modifier   string
	Precedence int
}

var NoAccess = access{"none", 0}
var ConstructorAccess = access{"constructor", 1}
var LocalAccess = access{"local", 2}
var TemplateAccess = access{"template", 3}

type accessibility struct {
	Modifier string
}

var NoAccessibility = accessibility{""}
var PublicAccessibility = accessibility{"public"}
var PrivateAccessibility = accessibility{"private"}
var ProtectedAccessibility = accessibility{"protected"}

type Definition struct {
	AccessModifier     accessibility
	Async              bool
	Decorators         []Decorator
	Generator          bool
	Getter             bool
	OriginFunctionName string
	IsAngularMethod    bool
	Name               string
	Node               *sitter.Node
	Override           bool
	Readonly           bool
	Setter             bool
	Static             bool
	UsageAccess        access
	Usages             []UsageInstance
}

func (def Definition) IsConstructorParam() bool {
	return def.OriginFunctionName == "constructor"
}

func (c Class) AddDefinition(definition Definition) Class {
	if definition.Usages == nil {
		definition.Usages = []UsageInstance{}
	}

	name := definition.Name

	definition.IsAngularMethod = IsAngularFunction(name)

	if c.Definitions == nil {
		c.Definitions = make(map[string]Definition)
	}
	c.Definitions[name] = definition

	return c
}

func (c Class) AppendDefinitionUsage(name string, usage UsageInstance) Class {
	definition, found := c.Definitions[name]
	if !found {
		return c
	}

	definition.UsageAccess = CalculateNewAccessType(definition.UsageAccess, usage.Access)
	definition.Usages = append(definition.Usages, usage)
	c.Definitions[name] = definition

	return c
}

func (c Class) AppendUsage(name string, usage UsageInstance) Class {
	usages, found := c.Usages[name]

	if found {
		usages.Usages = append(usages.Usages, usage)
		c.Usages[name] = usages

		return c
	}

	if !found {
		if c.Usages == nil {
			c.Usages = make(map[string]Usage)
		}

		c.Usages[name] = Usage{
			usage.Access,
			name,
			[]UsageInstance{usage},
		}
	}

	return c
}

func (c Class) SetUsageAccessType(name string, access access) Class {
	usage := c.Usages[name]
	usage.Access = CalculateNewAccessType(access, usage.Access)

	return c
}

type Decorator struct {
	IsAngular bool
	Name      string
}

type UsageInstance struct {
	Access access
	Node   *sitter.Node
}

type Usage struct {
	Access access
	Name   string
	Usages []UsageInstance
}

type Usages map[string]Usage

type Definitions map[string]Definition

func CalculateNewAccessType(new access, old access) access {
	if new.Precedence > old.Precedence {
		return new
	}

	return old
}

type State struct {
	Classes map[string]*Class
	Files   map[string]*File
	RootURI string
}

type File struct {
	Classes     [](*Class)
	Content     string
	Exports     [](*Reference)
	Filetype    string
	Imports     [](*ast.ImportParseResult)
	LineOffsets []uint32
	URI         string
	Version     int
}

type Class struct {
	AngularTemplateFile  *File
	Content              string
	Definitions          Definitions
	Extends              []*Reference
	ExtendsIdentNames    []string
	File                 *File
	Implements           []*Reference
	ImplementsIdentNames []string
	Name                 string
	Node                 *sitter.Node
	Usages               Usages
}

type Reference struct {
	Class *Class
	Name  string
	Node  *sitter.Node

	// Variable *Variable
	// ...
}

var logger = utils.GetLogger("parser_state")

func (c *Class) Postprocess(state *State) {
	c.resolveExtendsImplements(state)
}

func (c *Class) resolveExtendsImplements(state *State) {
	file := c.File

	extends := resolveIdentFromImports(c.ExtendsIdentNames, file, state)
	implements := resolveIdentFromImports(c.ImplementsIdentNames, file, state)

	c.Extends = extends
	c.Implements = implements
}

func getFileByPath(state *State, path string) *File {
	file, found := state.Files[path]
	if found {
		return file
	}

	if !utils.FileExists(path) {
		return nil
	}

	HandleFile(state, UriFromFilename(path), "", 0, "", logger)
	file, found = state.Files[path]
	if found {
		return file
	}

	return nil
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

func resolveNodeModulesImportPath(state *State, currentFile *File, importPath string) *File {
	currentPath := filepath.Dir(FilenameFromUri(currentFile.URI))

	for currentPath != "." && currentPath != "/" {
		nmPath := path.Join(currentPath, "node_modules")
		stat, err := os.Stat(nmPath)
		if err != nil {
			currentPath = path.Dir(currentPath)
			continue
		}

		if stat.IsDir() {
			resolvedFile := getFileByPath(state, path.Join(nmPath, importPath))
			if resolvedFile != nil {
				return resolvedFile
			}
		}

		currentPath = path.Dir(currentPath)
	}

	resolvedFile := getFileByPath(state, currentPath)
	if resolvedFile != nil {
		return resolvedFile
	}

	return nil
}

func resolveIdentFromImports(idents []string, file *File, state *State) []*Reference {
	resolved := make([]*Reference, len(idents))

	for identIndex, ident := range idents {
		importPath := file.FindImportPath(ident)

		if importPath == "" {
			continue
		}

		var importedFile *File

		extensions := []string{".ts", ".d.ts", ".js"}
		joinSuffixes := []string{"", "index"}

		for _, join := range joinSuffixes {
			ip := path.Join(importPath, join)
			for _, extension := range extensions {
				importedFile = resolveProjectImportPath(state, file, ip+extension)
				if importedFile != nil {
					break
				}
			}

			if importedFile == nil {
				for _, extension := range extensions {
					importedFile = resolveNodeModulesImportPath(state, file, ip+extension)
					if importedFile != nil {
						break
					}
				}
			}

			if importedFile != nil {
				break
			}
		}

		for _, export := range importedFile.Exports {
			if export.Name == ident {
				resolved[identIndex] = export
			}
		}
	}

	return resolved
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

func (c Class) Id() string {
	return c.File.URI + "-" + c.Name
}

func NewFile(uri string, filetype string, version int) (File, error) {
	filename := FilenameFromUri(uri)

	if !strings.HasPrefix(filename, "/") {
		cwd, err := os.Getwd()
		if err != nil {
			return File{}, err
		}

		filename, err = filepath.Abs(path.Join(cwd, filename)) // what?
		if err != nil {
			return File{}, err
		}
	}

	return File{
		Classes:     []*Class{},
		Content:     "",
		Exports:     []*Reference{},
		Filetype:    filetype,
		Imports:     []*ast.ImportParseResult{},
		LineOffsets: []uint32{},
		URI:         UriFromFilename(filename),
		Version:     version,
	}, nil
}

func (f File) Filename() string {
	return FilenameFromUri(f.URI)
}

func (f File) GetDependencies() []string {
	dependents := make([]string, 0)
	// for _, class := range f.Classes {
	// 	dependents = append(dependents, class.Controllers...)
	// 	if class.Template != "" {
	// 		dependents = append(dependents, class.Template)
	// 	}
	// }

	return dependents
}

func (c Class) GetGetters() []Definition {
	return filterDefinitions(c, func(d Definition) bool { return d.Getter })
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
		isLineStart = (ch == '\r' || ch == '\n')

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

func (f *File) SetContent(content string) {
	lineOffsets := getLineOffsets(content)
	f.LineOffsets = lineOffsets
	f.Content = content
}

// func (f File) SetContent(content string) File {
// 	lineOffsets := getLineOffsets(content)
// 	f.LineOffsets = lineOffsets
// 	f.Content = content
// 	return f
// }

// func (s State) GetClassForTemplate(uri string) *Class {
// 	for _, class := range s.Classes {
// 		if class.AngularTemplateFile != nil && class.AngularTemplateFile.URI == uri {
// 			return &class
// 		}
// 	}
//
// 	return nil
// }

func (f File) GetOffsetForPosition(p utils.Position) uint32 {
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

func (f File) GetOffsetsForRange(r utils.Range) (uint32, uint32) {
	return f.GetOffsetForPosition(r.Start), f.GetOffsetForPosition(r.End)
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
		if lineOffset >= offset {
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

func filterDefinitions(c Class, cond func(d Definition) bool) []Definition {
	arr := []Definition{}
	for _, definition := range c.Definitions {
		if cond(definition) {
			arr = append(arr, definition)
		}
	}

	return arr
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

func CalculateAccessibilityFromString(a string) (accessibility, error) {
	switch a {

	case "public":
		return PublicAccessibility, nil
	case "private":
		return PrivateAccessibility, nil
	case "protected":
		return ProtectedAccessibility, nil
	}

	return PublicAccessibility, fmt.Errorf("Unhandled accessibility: %s", a)
}

func CreatePropertyDefinition(accessModifier accessibility, decorators []Decorator, name string, node *sitter.Node) Definition {
	return Definition{
		AccessModifier:  accessModifier,
		Async:           false,
		Decorators:      decorators,
		Generator:       false,
		Getter:          false,
		IsAngularMethod: false,
		Name:            name,
		Node:            node,
		Override:        false,
		Readonly:        false,
		Setter:          false,
		Static:          false,
		UsageAccess:     access{},
		Usages:          []UsageInstance{},
	}
}
