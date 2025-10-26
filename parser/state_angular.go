package parser

import sitter "github.com/smacker/go-tree-sitter"

type Angular struct {
	Component *Component
	Module    *Module
}

type Component struct {
	DeclaredIn      *Class
	Imports         References
	ImportsIdents   []string
	Selector        string
	Template        *Template
	TemplateUrl     string
	TemplateUrlFile *File
}

type Module struct {
	Declarations       References
	DeclarationsIdents []string
	Exports            References
	ExportsIdents      []string
	Imports            References
	ImportsIdents      []string
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
}

func (c *Component) EnsureTemplate() {
	if c.Template == nil {
		c.Template = &Template{TagUsages: make(TagUsages, 0)}
	}
}

func (c *Component) Postprocess(state *State, class *Class) {
	imports := resolveIdentFromImports(c.ImportsIdents, class.File, state)
	c.Imports = imports
}

func (m *Module) Postprocess(state *State, class *Class) {
	m.Imports = resolveIdentFromImports(m.ImportsIdents, class.File, state)
	m.Exports = resolveIdentFromImports(m.ExportsIdents, class.File, state)
	m.Declarations = resolveIdentFromImports(m.DeclarationsIdents, class.File, state)

	for _, declaration := range m.Declarations {
		if declaration == nil {
			continue
		}

		if !declaration.Class.HasComponent() {
			continue
		}

		declaration.Class.Angular.Component.DeclaredIn = class
	}
}
