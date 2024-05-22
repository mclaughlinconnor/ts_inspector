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
	Usages         []UsageInstance
	UsageAccess    access
}

func (f File) AddDefinition(name string, definition Definition) File {
	if definition.Usages == nil {
		definition.Usages = []UsageInstance{}
	}

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
	Template    string
	URI         string
	Usages      Usages
	Version     int
}

func NewFile(uri string, filetype string, version int, controller string, template string) File {
	return File{
		"",            // Content
		controller,    // Controller
		Definitions{}, // Definitions
		filetype,      // Filetype
		template,      // Template
		uri,           // Uri
		Usages{},      // Usages
		version,       // Version
	}
}

func (f File) Filename() string {
	return FilenameFromUri(f.URI)
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
	return Definition{
		accessModifier,    // AccessModifier
		false,             // Async
		decorators,        // Decorators
		false,             // Generator
		false,             // Getter
		name,              // Name
		node,              // Node
		false,             // Override
		false,             // Readonly
		false,             // Setter
		false,             // Static
		[]UsageInstance{}, // Usages
		NoAccess,          // Usage access
	}
}
