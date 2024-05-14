package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type access struct {
	modifier   string
	precedence int
}

var ConstructorAccess = access{"constructor", 0}
var LocalAccess = access{"local", 1}
var ForeignAccess = access{"foreign", 2}

type accessibility struct {
	Modifier string
}

var PublicAccessibility = accessibility{"public"}
var PrivateAccessibility = accessibility{"private"}
var ProtectedAccessibility = accessibility{"protected"}

type Definition struct {
	AccessModifier accessibility
	Async          bool
	Decorators     []Decorator
	Generator      bool
	Getter         bool
	Name           string
	Node           *sitter.Node
	Override       bool
	Readonly       bool
	Setter         bool
	Static         bool
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
	Definitions Definitions
	Filetype    string
	Template    string
	URI         string
	Usages      Usages
	Version     int
}

func NewFile(uri string, filetype string, version int) File {
	return File{
		Definitions{},
		filetype,
		"",
		uri,
		Usages{},
		version,
	}
}

func (f File) GetGetters() []Definition {
	return filterDefinitions(f, func(d Definition) bool { return d.Getter })
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
	return Definition{accessModifier, false, decorators, false, false, name, node, false, false, false, false}
}
