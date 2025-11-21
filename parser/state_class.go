package parser

import (
	"cmp"
	"slices"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type Decorator struct {
	IsAngular bool
	Name      string
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
	NameNode             *sitter.Node
	Node                 *sitter.Node
	Usages               Usages
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

	if c.Usages == nil {
		c.Usages = make(map[string]Usage)
	}

	c.Usages[name] = Usage{usage.Access, name, []*UsageInstance{usage}}
}

func (c *Class) DropTemplateUsages() {
	for usageIndex, usage := range c.Usages {
		usageInstances := make([]*UsageInstance, 0)
		access := NoAccess

		for usageInstanceIndex := range usage.Usages {
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

	if c.HasComponent() && c.Angular.Component.Template != nil {
		c.Angular.Component.Template.TagUsages = make(TagUsages)
	}
}

func (c *Class) EnsureAngular() {
	if c.Angular == nil {
		c.Angular = &Angular{}
	}
}

type ClassedDefinition struct {
	Definition
	Class *Class
}

func (c *Class) FilterOwnDefinitions(cond func(d ClassedDefinition) bool) []ClassedDefinition {
	arr := []ClassedDefinition{}
	for _, definition := range c.GetClassedDefinitions() {
		if cond(definition) {
			arr = append(arr, definition)
		}
	}
	return arr
}

func (c *Class) FilterAllDefinitions(cond func(d ClassedDefinition) bool) []ClassedDefinition {
	definitions := c.FilterOwnDefinitions(cond)
	definitionsMap := make(map[string]bool)

	for _, d := range definitions {
		definitionsMap[d.Name] = true
	}

	for _, e := range c.Extends {
		if e == nil || e.Class == nil {
			continue
		}

		ds := e.Class.FilterOwnDefinitions(cond)
		for _, d := range ds {
			// Don't allow duplicates. Also, prepare for doing stuff with overridden props
			found, _ := definitionsMap[d.Name]
			if !found {
				definitionsMap[d.Name] = true
				definitions = append(definitions, d)
			}
		}
	}

	return definitions
}

func (c *Class) FilterAllDefinitionsByDecorator(decoratorName string) []ClassedDefinition {
	return c.FilterAllDefinitions(func(def ClassedDefinition) bool {
		return slices.ContainsFunc(def.Decorators, func(dec Decorator) bool { return dec.Name == decoratorName })
	})
}

func (c *Class) GetAllPublicDefinitions() []ClassedDefinition {
	definitions := c.GetOwnPublicDefinitions()
	definitionsMap := make(map[string]bool)

	for _, d := range definitions {
		definitionsMap[d.Name] = true
	}

	for _, e := range c.Extends {
		if e == nil || e.Class == nil {
			continue
		}

		ds := e.Class.GetAllPublicDefinitions()
		for _, d := range ds {
			// Don't allow duplicates. Also, prepare for doing stuff with overridden props
			found, _ := definitionsMap[d.Name]
			if !found {
				definitionsMap[d.Name] = true
				definitions = append(definitions, d)
			}
		}
	}

	return definitions
}

func (c *Class) GetClassedDefinitions() []ClassedDefinition {
	classedDefinitions := make([]ClassedDefinition, len(c.Definitions))

	i := 0
	for _, d := range c.Definitions {
		classedDefinitions[i] = ClassedDefinition{d, c}
		i++
	}

	return classedDefinitions
}

func (c *Class) GetDocumentation(includeClassName bool) string {
	documentation := make([]string, 0)

	if includeClassName {
		documentation = append(documentation, c.Name)
	}

	if c.HasComponent() && len(c.Angular.Component.DeclaredIn) > 0 {
		modules := make([]string, 0)

		for _, d := range c.Angular.Component.DeclaredIn {
			modules = append(modules, d.Name)
		}

		documentation = append(documentation, "Declared in: "+strings.Join(modules, ", "))
	}

	documentation = append(documentation, buildDefinitionSection("Inputs", c.GetInputs()))
	documentation = append(documentation, buildDefinitionSection("Outputs", c.GetOutputs()))

	return strings.Join(documentation, "\n\n")
}

func (c *Class) GetGetters() []ClassedDefinition {
	return c.FilterOwnDefinitions(func(d ClassedDefinition) bool { return d.Getter })
}

func (c *Class) GetInputs() []ClassedDefinition {
	return c.FilterAllDefinitionsByDecorator("Input")
}

func (c *Class) GetOutputs() []ClassedDefinition {
	return c.FilterAllDefinitionsByDecorator("Output")
}

func (c *Class) GetOwnPublicDefinitions() []ClassedDefinition {
	return c.FilterOwnDefinitions(func(d ClassedDefinition) bool { return d.IsPublic() })
}

func (c *Class) GetTemplateFile() *File {
	if c.Angular != nil && c.Angular.Component != nil {
		return c.Angular.Component.TemplateUrlFile
	}

	return nil
}

func (c *Class) HasComponent() bool {
	return c.Angular != nil && c.Angular.Component != nil
}

func (c *Class) HasDefinition(name string) bool {
	for _, d := range c.Definitions {
		if d.Name == name {
			return true
		}
	}

	return false
}

func (c *Class) HasModule() bool {
	return c.Angular != nil && c.Angular.Module != nil
}

func (c *Class) Id() string { return ClassId(c.File.URI, c.Name) }

func (c *Class) Postprocess(state *State) {
	c.resolveExtendsImplements(state)
	if c.Angular != nil {
		c.Angular.Postprocess(state, c)
	}
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

func (c *Class) SetUsageAccessType(name string, access access) {
	usage := c.Usages[name]
	usage.Access = CalculateNewAccessType(access, usage.Access)
}

func (c *Class) resolveExtendsImplements(state *State) {
	file := c.File
	extends := resolveIdentFromImports(c.ExtendsIdentNames, file, state)
	implements := resolveIdentFromImports(c.ImplementsIdentNames, file, state)
	c.Extends = extends
	c.Implements = implements
}

func ClassId(uri string, className string) string { return uri + "-" + className }

func NewClass(content string, file *File, node *sitter.Node) Class {
	return Class{Content: content, Definitions: make(map[string]Definition), Extends: []*Reference{}, ExtendsIdentNames: []string{}, File: file, Implements: []*Reference{}, ImplementsIdentNames: []string{}, Name: "", Node: node, Usages: make(map[string]Usage)}
}

func buildDefinitionSection(sectionName string, definitions []ClassedDefinition) string {
	section := make([]string, 0)

	slices.SortFunc(definitions, func(a ClassedDefinition, b ClassedDefinition) int { return cmp.Compare(a.Name, b.Name) })
	if len(definitions) > 0 {
		section = append(section, sectionName+":")
		for _, input := range definitions {
			section = append(section, "  "+input.Name+" ("+input.Class.Name+")")
		}
	}

	return strings.Join(section, "\n")
}

// For sorting by name
type Classes []*Class

func (c Classes) Len() int {
	return len(c)
}

func (c Classes) Less(a int, b int) bool {
	return c[a].Name < c[b].Name
}

func (c Classes) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
