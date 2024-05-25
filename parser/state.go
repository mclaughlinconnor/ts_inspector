package parser

import (
	"fmt"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type access struct {
	modifier   string
	precedence int
}

var NoAccess = access{"none", 0}
var ConstructorAccess = access{"constructor", 1}
var LocalAccess = access{"local", 2}
var ForeignAccess = access{"foreign", 3}

type accessibility struct {
	Modifier string
}

var PublicAccessibility = accessibility{"public"}
var PrivateAccessibility = accessibility{"private"}
var ProtectedAccessibility = accessibility{"protected"}

type Definition struct {
	AccessModifier  accessibility
	Async           bool
	Decorators      []Decorator
	Generator       bool
	Getter          bool
	Name            string
	Node            *sitter.Node
	Override        bool
	Readonly        bool
	Setter          bool
	Static          bool
	UsageAccess     access
	Usages          []UsageInstance
	IsAngularMethod bool
}

func (f File) AddDefinition(name string, definition Definition) File {
	if definition.Usages == nil {
		definition.Usages = []UsageInstance{}
	}

	definition.IsAngularMethod = IsAngularFunction(name)

	if f.Definitions == nil {
		f.Definitions = make(map[string]Definition)
	}
	f.Definitions[name] = definition

	return f
}

func (f File) AppendDefinitionUsage(name string, usage UsageInstance) File {
	definition, found := f.Definitions[name]
	if !found {
		return f
	}

	definition.UsageAccess = CalculateNewAccessType(definition.UsageAccess, usage.Access)
	definition.Usages = append(definition.Usages, usage)
	f.Definitions[name] = definition

	return f
}

func (f File) AppendUsage(name string, usage UsageInstance) File {
	usages, found := f.Usages[name]

	if found {
		usages.Usages = append(usages.Usages, usage)
		f.Usages[name] = usages

		return f
	}

	if !found {
		if f.Usages == nil {
			f.Usages = make(map[string]Usage)
		}

		f.Usages[name] = Usage{
			usage.Access,
			name,
			[]UsageInstance{usage},
		}
	}

	return f
}

func (f File) SetUsageAccessType(name string, access access) File {
	usage := f.Usages[name]
	usage.Access = CalculateNewAccessType(access, usage.Access)

	return f
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
	if new.precedence > old.precedence {
		return new
	}

	return old
}

type State map[string]File

type File struct {
	Content     string
	Controller  string
	Definitions Definitions
	Filetype    string
	LineOffsets []uint32
	Template    string
	URI         string
	Usages      Usages
	Version     int
}

func NewFile(uri string, filetype string, version int, controller string, template string) File {
	return File{
		Content:     "",
		Controller:  controller,
		Definitions: map[string]Definition{},
		Filetype:    filetype,
		LineOffsets: []uint32{},
		Template:    template,
		URI:         uri,
		Usages:      map[string]Usage{},
		Version:     version,
	}
}

func (f File) Filename() string {
	return FilenameFromUri(f.URI)
}

func (f File) GetGetters() []Definition {
	return filterDefinitions(f, func(d Definition) bool { return d.Getter })
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

func (f File) SetContent(content string) File {
	lineOffsets := getLineOffsets(content)
	f.LineOffsets = lineOffsets
	f.Content = content
	return f
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

func filterDefinitions(f File, cond func(d Definition) bool) []Definition {
	arr := []Definition{}
	for _, definition := range f.Definitions {
		if cond(definition) {
			arr = append(arr, definition)
		}
	}

	return arr
}

func FiletypeFromFilename(filename string) (string, error) {
	if strings.HasSuffix(filename, ".pug") {
		return "pug", nil
	} else if strings.HasSuffix(filename, ".pug") {
		return "typescript", nil
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
