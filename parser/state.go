package parser

import (
	"fmt"

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
	Usages      Usages
	Template    string
}

func NewFile() File {
	return File{
		Definitions{},
		Usages{},
		"",
	}
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
