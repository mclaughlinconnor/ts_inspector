package parser

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

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
	Type                 string
	UsageAccess          access
	Usages               []*UsageInstance
}

type Definitions map[string]Definition

type accessibility struct {
	Modifier string
}

var NoAccessibility = accessibility{""}
var PrivateAccessibility = accessibility{"private"}
var ProtectedAccessibility = accessibility{"protected"}
var PublicAccessibility = accessibility{"public"}

func (d *Definition) HasAngularDecorator() bool {
	for _, decorator := range d.Decorators {
		if decorator.IsAngular {
			return true
		}
	}

	return false
}

func (d *Definition) HasInjectDecorator() bool {
	for _, decorator := range d.Decorators {
		if decorator.Name == "Inject" {
			return true
		}
	}

	return false
}

func (d *Definition) IsAngularMethod() bool {
	return strings.HasPrefix(d.Name, "ng") && IsAngularFunction(d.Name)
}

func (d *Definition) IsConstructorParam() bool { return d.OriginFunctionName == "constructor" }
func (d *Definition) IsLocalParam() bool       { return d.AccessModifier == NoAccessibility }
func (d *Definition) IsPrivate() bool          { return d.AccessModifier == PrivateAccessibility }
func (d *Definition) IsProtected() bool        { return d.AccessModifier == ProtectedAccessibility }
func (d *Definition) IsPublic() bool           { return d.AccessModifier == PublicAccessibility }
func (d *Definition) IsUsed() bool             { return len(d.Usages) != 0 }

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

func CalculateNewAccessType(new access, old access) access {
	if new.Precedence > old.Precedence {
		return new
	}
	return old
}

func CreatePropertyDefinition(accessModifier accessibility, decorators []Decorator, name string, node *sitter.Node) Definition {
	return Definition{AccessModifier: accessModifier, Async: false, Decorators: decorators, Generator: false, Getter: false, IsAngularesqueMethod: false, Name: name, Node: node, Override: false, Readonly: false, Setter: false, Static: false, UsageAccess: access{}, Usages: []*UsageInstance{}}
}
