package parser

import (
	"slices"
	"sort"

	sitter "github.com/smacker/go-tree-sitter"
)

type Angular struct {
	Component *Component
	Module    *Module
}

type Component struct {
	DeclaredIn      []*Class
	Imports         *Value
	Providers       []*Provider
	Selectors       []string
	SelectorNode    *sitter.Node
	Template        *Template
	TemplateUrl     string
	TemplateUrlFile *File
}

type Module struct {
	Declarations *Value
	Exports      *Value
	Imports      *Value
	Providers    []*Provider
}

type Provider struct {
	Class    *Reference
	Existing *Reference
	Factory  *sitter.Node
	RefToken *Reference
	Token    *Reference
	Value    *sitter.Node
}

type TagUsage struct {
	// TODO
	// Args
	// Class  *Class

	Ident  string
	Usages []*UsageInstance
}

type TagUsages map[string]TagUsage

type Template struct {
	TagUsages TagUsages
}

func (a *Angular) DoesImport(class *Class) bool {
	if a.Component != nil {
		return a.Component.Imports.IsOrHas(class)
	}

	if a.Module != nil {
		return a.Module.Imports.IsOrHas(class)
	}

	return false
}

func (a *Angular) EnsureComponent() {
	if a.Component == nil {
		a.Component = &Component{}
	}
}

func (a *Angular) EnsureModule() {
	if a.Module == nil {
		a.Module = &Module{}
	}
}

func (a *Angular) Postprocess(state *State, class *Class) {
	if a.Module != nil {
		a.Module.Postprocess(state, class)
	}
}

func (c *Component) AddTagUsage(usageNode *sitter.Node, usageIdent string) {
	c.EnsureTemplate()

	usage, found := c.Template.TagUsages[usageIdent]
	if !found {
		c.Template.TagUsages[usageIdent] = TagUsage{Ident: usageIdent, Usages: []*UsageInstance{{Access: TemplateAccess, Node: usageNode}}}

		return
	}

	usageInstance := UsageInstance{Access: TemplateAccess, Node: usageNode}
	usage.Usages = append(usage.Usages, &usageInstance)

	c.Template.TagUsages[usageIdent] = usage
}

func (c *Component) EnsureTemplate() {
	if c.Template == nil {
		c.Template = &Template{TagUsages: make(TagUsages, 0)}
	}
}

func (c *Component) GetAvailableComponents(state *State) []*Class {
	selectors := make(Classes, 0)

	for _, declaringClass := range c.DeclaredIn {
		if declaringClass.Snapshot().Angular == nil || declaringClass.Snapshot().Angular.Module == nil {
			continue
		}

		selectors = append(selectors, declaringClass.Snapshot().Angular.Module.GetComponentsFromInside(state)...)
	}

	for imp := range c.Imports.FlattenReferenceArraysToReferences(state) {
		imp.Resolve(state)
		if imp == nil || imp.Class == nil {
			continue
		}

		if imp.Class.HasComponent() {
			selectors = append(selectors, imp.Class)
		}

		if imp.Class.HasModule() {
			selectors = append(selectors, imp.Class.Snapshot().Angular.Module.GetComponentsFromOutside(state)...)
		}
	}

	sort.Sort(selectors)

	return slices.Compact(selectors)
}

func (m *Module) DoesDeclare(class *Class) bool {
	return m.Declarations.IsOrHas(class)
}

func (m *Module) DoesExport(state *State, class *Class) bool {
	for exp := range m.Exports.FlattenReferenceArraysToReferences(state) {
		if exp == nil || exp.Class == nil {
			// Should have been resolved by now
			continue
		}

		if exp.Class == class {
			return true
		}
	}

	return false
}

func (m *Module) GetDeclaredComponents(state *State) []*Class {
	selectors := make([]*Class, 0)
	declarations := m.Declarations

	classes := []*Class{}
	if declarations.Type == "reference" {
		declarations.Reference.Resolve(state)

		class := declarations.Reference.Class
		if class != nil {
			classes = append(classes, class)
		}
	}

	if declarations.Type == "array" {
		for _, element := range declarations.ArrayValues {
			if element.Type != "reference" {
				continue
			}

			ref := element.Reference
			ref.Resolve(state)

			if ref.Class == nil {
				continue
			}

			classes = append(classes, ref.Class)
		}
	}

	for _, declaration := range classes {
		angular := declaration.Snapshot().Angular
		if angular == nil {
			continue
		}

		if angular.Component != nil {
			selectors = append(selectors, declaration)
		}

		// Illegal
		if angular.Module != nil {
			selectors = append(selectors, angular.Module.GetExportedComponents(state)...)
		}
	}

	return selectors
}

func (m *Module) GetExportedComponents(state *State) []*Class {
	selectors := make([]*Class, 0)

	for exp := range m.Exports.FlattenReferenceArraysToReferences(state) {
		if exp == nil || exp.Class == nil {
			// Should have been resolved by now
			continue
		}

		angular := exp.Class.Snapshot().Angular
		if angular == nil {
			continue
		}

		if angular.Component != nil {
			selectors = append(selectors, exp.Class)
		}

		if angular.Module != nil {
			selectors = append(selectors, angular.Module.GetImportedComponents(state)...)
		}
	}

	return selectors
}

func (m *Module) GetImportedComponents(state *State) []*Class {
	selectors := make([]*Class, 0)

	for imp := range m.Imports.FlattenReferenceArraysToReferences(state) {
		if imp == nil || imp.Class == nil {
			// Should have been resolved by now
			continue
		}

		angular := imp.Class.Snapshot().Angular
		if angular == nil {
			continue
		}

		if angular.Component != nil {
			selectors = append(selectors, imp.Class)
		}

		if angular.Module != nil {
			selectors = append(selectors, angular.Module.GetExportedComponents(state)...)
		}
	}

	return selectors
}

func (m *Module) GetComponentsFromInside(state *State) []*Class {
	selectors := m.GetDeclaredComponents(state)
	selectors = append(selectors, m.GetImportedComponents(state)...)

	return selectors
}

func (m *Module) GetComponentsFromOutside(state *State) []*Class {
	return m.GetExportedComponents(state)
}

func (m *Module) Postprocess(state *State, class *Class) {
	for declaration := range m.Declarations.FlattenReferenceArraysToReferences(state) {
		if declaration == nil {
			continue
		}

		declaration.Resolve(state)
		if declaration.Class == nil || !declaration.Class.HasComponent() {
			continue
		}

		/*
		 * Editing pug files can trigger a re-postprocess, which will add the same
		 * class without walk_typescript being able to call class.Reset()
		 */
		if !slices.Contains(declaration.Class.Snapshot().Angular.Component.DeclaredIn, class) {
			declaration.Class.Update(func(data *classState) {
				data.Angular.Component.DeclaredIn = append(data.Angular.Component.DeclaredIn, class)
			})
		}
	}
}
