package parser

import (
	"cmp"
	"maps"
	"slices"

	sitter "github.com/smacker/go-tree-sitter"
)

type Angular struct {
	Component *Component
	Directive *Directive
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

type Directive struct {
	DeclaredIn   []*Class
	Imports      *Value
	Providers    []*Provider
	Selectors    []string
	SelectorNode *sitter.Node
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

// TODO: this string should probably be ast.Tag so that I can handle selectors like `li[cmData]`
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

func (a *Angular) EnsureDirective() {
	if a.Directive == nil {
		a.Directive = &Directive{}
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

func (c *Component) GetAvailableThings(state *State) []*Class {
	things := make(map[string]*Class)

	for _, declaringClass := range c.DeclaredIn {
		if declaringClass.Snapshot().Angular == nil || declaringClass.Snapshot().Angular.Module == nil {
			continue
		}

		for _, thing := range declaringClass.Snapshot().Angular.Module.GetThingsFromInside(state) {
			things[thing.Id()] = thing
		}

	}

	for imp := range c.Imports.FlattenReferenceArraysToReferences(state) {
		if imp == nil || imp.Class == nil {
			continue
		}

		if imp.Class.HasComponent() || imp.Class.HasDirective() {
			things[imp.Class.Id()] = imp.Class
		}

		if imp.Class.HasModule() {
			for _, thing := range imp.Class.Snapshot().Angular.Module.GetThingsFromOutside(state) {
				things[thing.Id()] = thing
			}
		}
	}

	vs := slices.Collect(maps.Values(things))
	slices.SortFunc(vs, func(a *Class, b *Class) int { return cmp.Compare(b.Snapshot().Name, a.Snapshot().Name) })

	return vs
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

func (m *Module) GetDeclaredThings(state *State) []*Class {
	things := make([]*Class, 0)
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

		if angular.Component != nil || angular.Directive != nil {
			things = append(things, declaration)
		}

		// Illegal
		if angular.Module != nil {
			things = append(things, angular.Module.GetExportedThings(state)...)
		}
	}

	return things
}

func (m *Module) GetExportedThings(state *State) []*Class {
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

		if angular.Component != nil || angular.Directive != nil {
			selectors = append(selectors, exp.Class)
		}

		if angular.Module != nil {
			selectors = append(selectors, angular.Module.GetImportedThings(state)...)
		}
	}

	return selectors
}

func (m *Module) GetImportedThings(state *State) []*Class {
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

		if angular.Component != nil || angular.Directive != nil {
			selectors = append(selectors, imp.Class)
		}

		if angular.Module != nil {
			selectors = append(selectors, angular.Module.GetExportedThings(state)...)
		}
	}

	return selectors
}

func (m *Module) GetThingsFromInside(state *State) []*Class {
	selectors := m.GetDeclaredThings(state)
	selectors = append(selectors, m.GetImportedThings(state)...)

	return selectors
}

func (m *Module) GetThingsFromOutside(state *State) []*Class {
	return m.GetExportedThings(state)
}

func (m *Module) Postprocess(state *State, class *Class) {
	for declaration := range m.Declarations.FlattenReferenceArraysToReferences(state) {
		if declaration.Class == nil {
			continue
		}

		/*
		 * Editing pug files can trigger a re-postprocess, which will add the same
		 * class without walk_typescript being able to call class.Reset()
		 */

		if declaration.Class.HasComponent() && !slices.Contains(declaration.Class.Snapshot().Angular.Component.DeclaredIn, class) {
			declaration.Class.Update(func(data *classState) {
				data.Angular.Component.DeclaredIn = append(data.Angular.Component.DeclaredIn, class)
			})
		}

		if declaration.Class.HasDirective() && !slices.Contains(declaration.Class.Snapshot().Angular.Directive.DeclaredIn, class) {
			declaration.Class.Update(func(data *classState) {
				data.Angular.Directive.DeclaredIn = append(data.Angular.Directive.DeclaredIn, class)
			})
		}
	}
}
