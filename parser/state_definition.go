package parser

import (
	"fmt"
	"iter"
	"maps"
	"strings"
	"sync"
	"ts_inspector/utils"

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

type Definitions struct {
	sync.RWMutex

	data map[string]*Definition
}

type accessibility struct {
	Modifier string
}

var NoAccessibility = accessibility{""}
var PrivateAccessibility = accessibility{"private"}
var ProtectedAccessibility = accessibility{"protected"}
var PublicAccessibility = accessibility{"public"}

func (d *Definitions) All() iter.Seq2[string, Definition] {
	data := maps.Clone(d.data)

	return func(yield func(string, Definition) bool) {
		for i, v := range data {
			if !yield(i, *v) {
				return
			}
		}
	}
}

func (d *Definitions) Clear() {
	clear(d.data)
}

func (d *Definitions) Get(name string) (Definition, bool) {
	d.RLock()
	definition, found := d.data[name]
	d.RUnlock()

	if found {
		return *definition, true
	}

	return Definition{}, false
}

func (d *Definitions) Len() int {
	return len(d.data)
}

func (d *Definitions) Set(name string, definition Definition) {
	d.Lock()
	d.data[name] = &definition
	d.Unlock()
}

func (d *Definition) GetDocumentation(includeDefinitionName bool) string {
	documentation := make([]string, 0)

	if includeDefinitionName {
		documentation = append(documentation, "# "+d.Name)
		documentation = append(documentation, "")
	}

	signature := make([]string, 0)
	if d.AccessModifier != NoAccessibility {
		signature = append(signature, d.AccessModifier.Modifier)
	}
	if d.Static {
		signature = append(signature, "static")
	}

	if d.Override {
		signature = append(signature, "override")
	}
	if d.Readonly {
		signature = append(signature, "readonly")
	}

	if d.Getter {
		signature = append(signature, "get")
	}
	if d.Setter {
		signature = append(signature, "setter")
	}

	name := d.Name
	if d.Generator {
		name = "*" + name
	}

	nodeType := d.Node.Type()
	isMethod := nodeType == "method_definition" || nodeType == "method_signature" || nodeType == "abstract_method_signature"
	if isMethod {
		name += "()"
	}

	name += ": " + d.Type

	signature = append(signature, name)

	documentation = append(documentation, "```ts\n"+strings.Join(signature, " ")+"\n```")

	firstLine := false

	if len(d.Decorators) > 0 {
		decorators := make([]string, 0)
		decorators = append(decorators, "**Decorators:**")
		for _, dec := range d.Decorators {
			str := "  `@" + dec.Name
			if len(dec.Arguments) > 0 {
				str += "("
				sb := make([]string, 0)
				sb = append(sb, dec.Arguments...)
				str += strings.Join(sb, ", ") + ")"
			} else {
				str += "()"
			}

			str += "`"
			if dec.IsAngular {
				str += " (Angular)"
			}

			decorators = append(decorators, str)
		}

		if !firstLine {
			documentation = append(documentation, "")
			firstLine = true
		}

		documentation = append(documentation, strings.Join(decorators, "\n"))
	}

	if isMethod {
		var angular string
		if d.IsAngularesqueMethod {
			angular = "true"
		} else {
			angular = "false"
		}

		if !firstLine {
			documentation = append(documentation, "")
		}

		documentation = append(documentation, "**Is Angular Method:** "+angular)
	}

	return strings.Join(documentation, "\n")
}

func (d *Definition) GetInputName() string {
	return d.getDefinitionNameByDecoratorArg("Input")
}

func (d *Definition) GetNameNode() *sitter.Node {
	nameNode := d.Node.ChildByFieldName("name")

	if nameNode != nil {
		return nameNode
	}

	return d.Node
}

func (d *Definition) GetOutputName() string {
	return d.getDefinitionNameByDecoratorArg("Output")
}

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

func (d *Definition) NameMatchesString(name string) bool {
	stripped, mode := utils.StripAngularFromAttribute(name)
	if mode == utils.NeitherAngularStripped || mode == utils.StructuralStripped {
		return d.Name == stripped
	}

	if (mode & utils.InputAngularStripped) > 0 {
		if d.GetInputName() == stripped {
			return true
		}
	}

	if (mode & utils.OutputAngularStripped) > 0 {
		if d.GetOutputName() == stripped {
			return true
		}
	}

	return false
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
	return PublicAccessibility, fmt.Errorf("unhandled accessibility: %s", a)
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

func (d *Definition) getDefinitionNameByDecoratorArg(decoratorName string) string {
	for _, decorator := range d.Decorators {
		if decorator.Name != decoratorName {
			continue
		}
		if len(decorator.Arguments) != 1 {
			return d.Name // There can't be two @Outputs or @Inputs on one prop
		}

		arg := decorator.Arguments[0]
		hasDoubleQuote := strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"")
		hasSingleQuote := strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'")
		if !hasDoubleQuote && !hasSingleQuote {
			return d.Name
		}

		quote := "'"
		if hasDoubleQuote {
			quote = "\""
		}

		return strings.TrimPrefix(strings.TrimSuffix(arg, quote), quote)
	}

	return d.Name
}
