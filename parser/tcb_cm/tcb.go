package tcb_cm

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"ts_inspector/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

type Import struct {
	Class      *parser.Class
	File       *parser.File
	Identifier string
}

type Tcb struct {
	CurrentScope *Scope
	Imports      []*Import
	NextId       int
	RootScope    *Scope
	State        *parser.State
	Class        *parser.Class
}

func (t *Tcb) GetNextId() int {
	id := t.NextId
	t.NextId += 1

	return id
}

func (t *Tcb) GetNextIdString() string {
	id := t.NextId
	t.NextId += 1

	return strconv.Itoa(id)
}

func (t *Tcb) AddAssignment(identifer string, identNode *sitter.Node, value StatementParts) {
	if identNode == nil {
		t.AddVirtPart(identifer)
	} else {
		t.AddRealPart(identifer, identNode)
	}

	t.AddVirtPart(" = ")
	t.AddStatementParts(&value)
	t.AddVirtPart(";\n")
}

func (t *Tcb) AddImport(class *parser.Class) string {
	f := class.Snapshot().File
	uri := f.Snapshot().URI

	var i *Import

	index := slices.IndexFunc(t.Imports, func(i *Import) bool { return i.File.Snapshot().URI == uri })
	if index != -1 {
		i = t.Imports[index]
	} else {
		i = &Import{Class: class, File: f, Identifier: t.GetNextIdString()}
		t.Imports = append(t.Imports, i)
	}

	return "i" + i.Identifier + "." + class.Snapshot().Name
}

func (t *Tcb) AddPart(part Part) {
	t.GetScope().AddPart(part)
}

func (t *Tcb) AddVirtPart(part string) {
	t.GetScope().AddVirtPart(part)
}

func (t *Tcb) AddStatementParts(parts *StatementParts) {
	t.GetScope().AddStatementParts(parts)
}

func (t *Tcb) AddRealPart(part string, node *sitter.Node) {
	t.GetScope().AddRealPart(part, node)
}

func (t *Tcb) BeginScope() {
	t.NewScope()
	t.AddVirtPart("{\n")
}

func (t *Tcb) BuildImports() string {
	sb := strings.Builder{}

	path := t.Class.Snapshot().Angular.Component.TemplateUrlFile.Filename()

	for _, i := range t.Imports {
		ifname := i.File.Filename()
		relative, _ := filepath.Rel(filepath.Dir(path), ifname)
		relativePath := "./" + strings.TrimSuffix(relative, filepath.Ext(relative))

		fmt.Fprintf(&sb, "import * as i%s from '%s';\n", i.Identifier, relativePath)
	}

	return sb.String()
}

func (t *Tcb) CreateVar(value StatementParts, identNode *sitter.Node) string {
	if v := t.GetScope().GetVariable(value); v != nil {
		return v.Identifier
	}

	name := "_t" + t.GetNextIdString()
	t.AddVirtPart("var ")
	t.AddRealPart(name, identNode)
	t.AddVirtPart(" = ")

	for _, p := range value.Parts {
		t.AddPart(p)
	}

	t.AddVirtPart(";\n")

	t.GetScope().AddVariable(&Variable{Identifier: name, Value: value.ToString()})

	return name
}

func (t *Tcb) EndScope() {
	t.AddVirtPart("\n}")
	t.CurrentScope = t.CurrentScope.ParentScope
}

func (t *Tcb) GetScope() *Scope {
	return t.CurrentScope
}

func (t *Tcb) NewScope() {
	scope := Scope{ParentScope: t.CurrentScope, Parts: StatementParts{}, Variables: []*Variable{}}
	t.CurrentScope.ChildScope = &scope
	t.CurrentScope = &scope
}

func (t *Tcb) ToString() *StatementParts {
	tcb := StatementParts{}

	tcb.AddVirtPart(t.BuildImports())

	scope := t.RootScope
	for scope != nil {
		tcb.AddStatementParts(&scope.Parts)
		scope = scope.ChildScope
	}

	return &tcb
}

func (t *Tcb) WithScope(builder func()) {
	t.BeginScope()
	builder()
	t.EndScope()
}

func GenerateTcb(state *parser.State, template *parser.Class, root *sitter.Node, content []byte) *StatementParts {
	scope := &Scope{}

	tcb := Tcb{
		CurrentScope: scope,
		NextId:       0,
		RootScope:    scope,
		State:        state,
		Class:        template,
	}

	ast := Parse(root, content, &tcb)

	buildTemplatePreamble(&tcb)

	tcb.WithScope(func() {
		ast.Render()
	})

	return tcb.ToString()
}

func InitTcb() {
	initTcbExpression()
	initAstParser()
}

func buildTemplatePreamble(tcb *Tcb) {
	tcb.GetScope().AddVirtPart("function _tcb")
	tcb.GetScope().AddVirtPart(tcb.GetNextIdString())
	tcb.GetScope().AddVirtPart("(this: ")

	classIdent := tcb.AddImport(tcb.Class)
	tcb.GetScope().AddVirtPart(classIdent)

	tcb.GetScope().AddVirtPart(") ")
}
