package parser

type Angular struct {
	Component *Component
	Module    *Module
}

type Component struct {
	Imports         References
	ImportsIdents   []string
	Selector        string
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

func (c *Component) Postprocess(state *State, class *Class) {
	imports := resolveIdentFromImports(c.ImportsIdents, class.File, state)
	c.Imports = imports
}

func (m *Module) Postprocess(state *State, class *Class) {
	m.Imports = resolveIdentFromImports(m.ImportsIdents, class.File, state)
	m.Exports = resolveIdentFromImports(m.ExportsIdents, class.File, state)
	m.Declarations = resolveIdentFromImports(m.DeclarationsIdents, class.File, state)
}
