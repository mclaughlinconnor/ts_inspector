package parser

import (
	"slices"
	"sort"
	"sync"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Angular struct {
	Component *Component
	Module    *Module
}

type Component struct {
	DeclaredIn      []*Class
	Imports         References
	ImportsIdents   []string
	Providers       []*Provider
	Selectors       []string
	Template        *Template
	TemplateUrl     string
	TemplateUrlFile *File
}

type Module struct {
	Declarations           References
	DeclarationsIdents     []string
	DeclarationsIdentNodes []*sitter.Node // Note: These are file based nodes
	Exports                References
	ExportsIdents          []string
	ExportsIdentNodes      []*sitter.Node // Note: These are file based nodes
	Imports                References
	ImportsIdents          []string
	ImportsIdentNodes      []*sitter.Node // Note: These are file based nodes
	Providers              []*Provider
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
	if a.Component != nil {
		a.Component.Postprocess(state, class)
	}

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

func (c *Component) GetAvailableComponents() []*Class {
	selectors := make(Classes, 0)

	for _, declaringClass := range c.DeclaredIn {
		if declaringClass.Snapshot().Angular == nil || declaringClass.Snapshot().Angular.Module == nil {
			continue
		}

		selectors = append(selectors, declaringClass.Snapshot().Angular.Module.GetComponentsFromInside()...)
	}

	for _, imp := range c.Imports {
		if imp == nil || imp.Class == nil {
			continue
		}

		if imp.Class.HasComponent() {
			selectors = append(selectors, imp.Class)
		}

		if imp.Class.HasModule() {
			selectors = append(selectors, imp.Class.Snapshot().Angular.Module.GetComponentsFromOutside()...)
		}
	}

	sort.Sort(selectors)

	return slices.Compact(selectors)
}

func (c *Component) Postprocess(state *State, class *Class) {
	imports := resolveIdentFromImports(c.ImportsIdents, class.Snapshot().File, state)
	c.Imports = imports
}

func (m *Module) DoesDeclare(class *Class) bool {
	for _, exp := range m.Declarations {
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

func (m *Module) DoesExport(class *Class) bool {
	for _, exp := range m.Exports {
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

func (m *Module) GetDeclaredComponents() []*Class {
	selectors := make([]*Class, 0)

	for _, imp := range m.Declarations {
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

		// Illegal
		if angular.Module != nil {
			selectors = append(selectors, angular.Module.GetExportedComponents()...)
		}
	}

	return selectors
}

func (m *Module) GetExportedComponents() []*Class {
	selectors := make([]*Class, 0)

	for _, exp := range m.Exports {
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
			selectors = append(selectors, angular.Module.GetImportedComponents()...)
		}
	}

	return selectors
}

func (m *Module) GetImportedComponents() []*Class {
	selectors := make([]*Class, 0)

	for _, imp := range m.Imports {
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
			selectors = append(selectors, angular.Module.GetExportedComponents()...)
		}
	}

	return selectors
}

func (m *Module) GetComponentsFromInside() []*Class {
	selectors := m.GetDeclaredComponents()
	selectors = append(selectors, m.GetImportedComponents()...)

	return selectors
}

func (m *Module) GetComponentsFromOutside() []*Class {
	return m.GetExportedComponents()
}

func (m *Module) Postprocess(state *State, class *Class) {
	wg := sync.WaitGroup{}

	wg.Go(func() { m.Imports = resolveIdentFromImports(m.ImportsIdents, class.Snapshot().File, state) })
	if !utils.Concurrency {
		wg.Wait()
	}

	wg.Go(func() { m.Exports = resolveIdentFromImports(m.ExportsIdents, class.Snapshot().File, state) })
	if !utils.Concurrency {
		wg.Wait()
	}

	wg.Go(func() { m.Declarations = resolveIdentFromImports(m.DeclarationsIdents, class.Snapshot().File, state) })
	if !utils.Concurrency {
		wg.Wait()
	}

	wg.Wait()

	for _, declaration := range m.Declarations {
		if declaration == nil {
			continue
		}

		if !declaration.Class.HasComponent() {
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
