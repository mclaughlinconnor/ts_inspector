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
	AccessModifier       accessibility
	Async                bool
	Decorators           []Decorator
	Generator            bool
	Getter               bool
	OriginFunctionName   string
	IsAngularesqueMethod bool
	Name                 string
	Node                 *sitter.Node
	Override             bool
	Readonly             bool
	Setter               bool
	Static               bool
	UsageAccess          access
	Usages               []*UsageInstance
}

func (def Definition) IsConstructorParam() bool {
	return def.OriginFunctionName == "constructor"
}

func (d *Definition) IsAngularMethod() bool {
	return strings.HasPrefix(d.Name, "ng") && IsAngularFunction(d.Name)
}

func (c *Class) AddDefinition(definition Definition) {
	if definition.Usages == nil {
		definition.Usages = []*UsageInstance{}
	}

	name := definition.Name

	definition.IsAngularesqueMethod = IsAngularFunction(name)

	if c.Definitions == nil {
		c.Definitions = make(map[string]Definition)
	}
	c.Definitions[name] = definition
}

func (c *Class) DropTemplateUsages() {
	for usageIndex, usage := range c.Usages {
		usageInstances := make([]*UsageInstance, 0)
		access := NoAccess
		for usageInstanceIndex, _ := range usage.Usages {
			usageInstance := usage.Usages[usageInstanceIndex]

			if usageInstance.Access != TemplateAccess {
				usageInstances = append(usageInstances, usageInstance)
				access = CalculateNewAccessType(access, usageInstance.Access)
			}
		}

		usage.Usages = usageInstances
		usage.Access = access

		c.Usages[usageIndex] = usage

		for definitionIndex, definition := range c.Definitions {
			if definition.Name == usage.Name {
				definition.Usages = usageInstances
				definition.UsageAccess = access
			}

			c.Definitions[definitionIndex] = definition
		}
	}

	for key, usage := range c.Usages {
		if len(usage.Usages) == 0 {
			delete(c.Usages, key)
		}
	}
}

func (c *Class) AppendDefinitionUsage(name string, usage *UsageInstance) {
	definition, found := c.Definitions[name]
	if !found {
		return
	}

	definition.UsageAccess = CalculateNewAccessType(definition.UsageAccess, usage.Access)
	definition.Usages = append(definition.Usages, usage)
	c.Definitions[name] = definition
}

func (c *Class) AppendUsage(name string, usage *UsageInstance) {
	usages, found := c.Usages[name]

	if found {
		usages.Usages = append(usages.Usages, usage)
		c.Usages[name] = usages

		return
	}

	if !found {
		if c.Usages == nil {
			c.Usages = make(map[string]Usage)
		}

		c.Usages[name] = Usage{
			usage.Access,
			name,
			[]*UsageInstance{usage},
		}
	}
}

func (c *Class) SetUsageAccessType(name string, access access) {
	usage := c.Usages[name]
	usage.Access = CalculateNewAccessType(access, usage.Access)
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
	Usages []*UsageInstance
}

type Usages map[string]Usage

type Definitions map[string]Definition

func (d *Definition) IsLocalParam() bool {
	return d.AccessModifier == NoAccessibility
}

func (d *Definition) IsUsed() bool {
	return len(d.Usages) != 0
}

func (d *Definition) HasAngularDecorator() bool {
	for _, decorator := range d.Decorators {
		if decorator.IsAngular {
			return true
		}
	}

	return false
}

func (d *Definition) IsPublic() bool {
	return d.AccessModifier == PublicAccessibility
}

func (d *Definition) IsPrivate() bool {
	return d.AccessModifier == PrivateAccessibility
}

func (d *Definition) IsProtected() bool {
	return d.AccessModifier == ProtectedAccessibility
}

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
	Exports     References
	Filetype    string
	Imports     [](*ast.ImportParseResult)
	LineOffsets []uint32
	URI         string
	Version     int
}

func (f *File) Postprocess(state *State) {
  for _, class := range f.Classes {
    state.Classes[class.Id()] = class

    class.Postprocess(state)
  }
}

func (f *File) ResetClasses() {
	f.Classes = make([](*Class), 0)
	f.Exports = make(References, 0)
}

type Component struct {
	Imports         References
	ImportsIdents   []string
	Selector        string
	TemplateUrl     string
	TemplateUrlFile *File
}

type Angular struct {
	Component *Component
}

type Class struct {
	Angular              *Angular
	Content              string
	Definitions          Definitions
	Extends              References // Extends may have nil references if resolution failed
	ExtendsIdentNames    []string
	File                 *File
	Implements           References // Implements may have nil references if resolution failed
	ImplementsIdentNames []string
	Name                 string
	Node                 *sitter.Node
	Usages               Usages
}

func (c *Component) Postprocess(state *State, class *Class) {
	imports := resolveIdentFromImports(c.ImportsIdents, class.File, state)
	c.Imports = imports
}

func (a *Angular) Postprocess(state *State, class *Class) {
	if a.Component != nil {
		a.Component.Postprocess(state, class)
	}
}

func (a *Angular) EnsureComponent() {
	if a.Component == nil {
		a.Component = &Component{}
	}
}

func (c *Class) GetTemplateFile() *File {
	if c.Angular != nil && c.Angular.Component != nil {
		return c.Angular.Component.TemplateUrlFile
	}

	return nil
}

func (c *Class) EnsureAngular() {
	if c.Angular == nil {
		c.Angular = &Angular{}
	}
}

type References []*Reference

func (r *References) IterateResolved(yield func(*Reference) bool) {
	for _, v := range *r {
		if v == nil {
			continue
		}

		if !yield(v) {
			return
		}
	}
}

type Reference struct {
	Class *Class
	Name  string
	Node  *sitter.Node

	// Variable *Variable
	// ...
}

func NewClass(content string, file *File, node *sitter.Node) Class {
	return Class{
		Content:              content,
		Definitions:          make(map[string]Definition),
		Extends:              []*Reference{},
		ExtendsIdentNames:    []string{},
		File:                 file,
		Implements:           []*Reference{},
		ImplementsIdentNames: []string{},
		Name:                 "",
		Node:                 node,
		Usages:               make(map[string]Usage),
	}
}

var logger = utils.GetLogger("parser_state")

func (c *Class) Postprocess(state *State) {
	c.resolveExtendsImplements(state)

	if c.Angular != nil {
		c.Angular.Postprocess(state, c)
	}
}

func (c *Class) HasDefinition(name string) bool {
	for _, d := range c.Definitions {
		if d.Name == name {
			return true
		}
	}

	return false
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

	IndexFileFromIndexer(state, path)
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

func (c *Class) Id() string {
	return ClassId(c.File.URI, c.Name)
}

func ClassId(uri string, className string) string {
	return uri + "-" + className
}

// Clears everything except the reference to the parent file
func (c *Class) Reset() {
	c.Angular = nil
	c.Content = ""
	clear(c.Definitions)
	clear(c.Extends)
	c.ExtendsIdentNames = make([]string, 0)
	clear(c.Implements)
	c.ImplementsIdentNames = make([]string, 0)
	c.Name = ""
	c.Node = nil
	clear(c.Usages)
}

func NewFile(uri string, filetype string, version int) (*File, error) {
	filename := FilenameFromUri(uri)

	if !strings.HasPrefix(filename, "/") {
		cwd, err := os.Getwd()
		if err != nil {
			file := File{}
			return &file, err
		}

		filename, err = filepath.Abs(path.Join(cwd, filename)) // what?
		if err != nil {
			file := File{}
			return &file, err
		}
	}

	file := File{
		Classes:     []*Class{},
		Content:     "",
		Exports:     []*Reference{},
		Filetype:    filetype,
		Imports:     []*ast.ImportParseResult{},
		LineOffsets: []uint32{},
		URI:         UriFromFilename(filename),
		Version:     version,
	}

	return &file, nil
}

func (f File) Filename() string {
	return FilenameFromUri(f.URI)
}

func (f *File) GetDependencies(state *State) []string {
	dependents := make([]string, 0)

	for _, class := range state.Classes {
		if class.GetTemplateFile() == f {
			dependents = append(dependents, class.File.Filename())
		}
	}

	for _, class := range f.Classes {
		t := class.GetTemplateFile()
		if t == f {
			dependents = append(dependents, t.Filename())
		}

		if class.File.Filename() != f.Filename() {
			dependents = append(dependents, class.File.Filename())
		}
	}

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

var lastSeenDocumentVersion = make(map[string]int, 0)

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

func (f *File) SetContent(content string, version int) {
	lineOffsets := getLineOffsets(content)
	f.LineOffsets = lineOffsets
	f.Content = content

	f.Version = versionFallback(version, f.URI)
}

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
		AccessModifier:       accessModifier,
		Async:                false,
		Decorators:           decorators,
		Generator:            false,
		Getter:               false,
		IsAngularesqueMethod: false,
		Name:                 name,
		Node:                 node,
		Override:             false,
		Readonly:             false,
		Setter:               false,
		Static:               false,
		UsageAccess:          access{},
		Usages:               []*UsageInstance{},
	}
}
