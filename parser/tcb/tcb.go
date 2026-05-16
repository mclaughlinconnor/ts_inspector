package tcb

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Directive struct {
	Class         *parser.Class
	Identifier    string
	IsConstructor bool
	Statement     *Statement
}

type GenericDirectiveConstructor struct {
	class      *parser.Class
	identifier string
	statement  *Statement
}

type Import struct {
	Class      *parser.Class
	File       *parser.File
	Identifier string
}

type Tcb struct {
	Ast                          *Ast
	Class                        *parser.Class
	CurrentScope                 *Scope
	Directives                   []*Directive
	GenericDirectiveConstructors []*GenericDirectiveConstructor
	Imports                      []*Import
	NextId                       int
	RootScope                    *Scope
	State                        *parser.State
	TagBoundaryPartStack         utils.Stack[*Part]
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

func (t *Tcb) AddAssignment(identifer string, identNode *sitter.Node, value Statement) {
	if identNode == nil {
		t.AddVirtPart(identifer)
	} else {
		t.AddRealPart(identifer, identNode)
	}

	t.AddVirtPart(" = ")
	t.AddStatement(&value)
	t.AddVirtPart(";\n")
}

func (t *Tcb) AddDirectiveConstructor(identifer string, directive *parser.Class, statement *Statement, isConstructor bool) {
	constructor := Directive{Class: directive, Identifier: identifer, IsConstructor: isConstructor, Statement: statement}
	t.Directives = append(t.Directives, &constructor)
}

func (t *Tcb) AddGenericDirectiveConstructor(directive *parser.Class, parts *Statement) string {
	index := slices.IndexFunc(t.GenericDirectiveConstructors, func(c *GenericDirectiveConstructor) bool { return c.class.Id() == directive.Id() })
	if index != -1 {
		return t.GenericDirectiveConstructors[index].identifier
	}

	g := GenericDirectiveConstructor{class: directive, statement: parts, identifier: "_t" + t.GetNextIdString()}
	t.GenericDirectiveConstructors = append(t.GenericDirectiveConstructors, &g)

	return g.identifier
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

func (t *Tcb) AddStatement(parts *Statement) {
	t.GetScope().AddStatement(parts)
}

func (t *Tcb) AddStatementAfterPart(parts *Statement, after *Part) *Part {
	return t.GetScope().AddStatementAfterPart(parts, after)
}

func (t *Tcb) AddRealPart(part string, node *sitter.Node) {
	t.GetScope().AddRealPart(part, node)
}

func (t *Tcb) BeginScope() {
	t.NewScope()
	t.AddVirtPart("{\n")
}

func (t *Tcb) BuildDirectiveConstructorsStatement() *Statement {
	statement := Statement{}

	for _, directive := range t.Directives {
		if !directive.IsConstructor {
			continue
		}

		statement.AddStatement(directive.Statement)
	}

	return &statement
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

func (t *Tcb) BuildGenericConstructors() string {
	sb := strings.Builder{}

	for _, constructor := range t.GenericDirectiveConstructors {
		sb.WriteString(constructor.statement.ToString())
	}

	return sb.String()
}

func (t *Tcb) CreateVarInCurrentScope(value *Statement) string {
	return t.CreateVarInScope(value, t.GetScope())
}

func (t *Tcb) CreateVarInRootScope(value *Statement) string {
	// TODO: should probably put the var at the top of the scope, not at the end
	return t.CreateVarInScope(value, t.RootScope)
}

func (t *Tcb) CreateVarInScope(value *Statement, scope *Scope) string {
	if v := scope.GetVariableByValue(value); v != nil {
		return v.Identifier
	}

	name := "_t" + t.GetNextIdString()
	scope.AddVirtPart("var ")
	scope.AddVirtPart(name)
	scope.AddVirtPart(" = ")

	scope.AddStatement(value)

	scope.AddVirtPart(";\n")

	lastPart := t.CurrentScope.Parts.GetLastPart()
	scope.AddVariable(&Variable{Identifier: name, LastPart: lastPart, Value: value.ToString()})

	return name
}

func (t *Tcb) CreateVarAfterPart(value *Statement, after *Part) (string, *Part) {
	if v := t.GetScope().GetVariableByValue(value); v != nil {
		return v.Identifier, v.LastPart
	}

	name := "_t" + t.GetNextIdString()
	statement := Statement{}
	statement.AddVirtPart("var ")
	statement.AddVirtPart(name)
	statement.AddVirtPart(" = ")

	statement.AddStatement(value)

	statement.AddVirtPart(";\n")

	newAfter := t.AddStatementAfterPart(&statement, after)

	lastPart := t.CurrentScope.Parts.GetLastPart()
	t.GetScope().AddVariable(&Variable{Identifier: name, LastPart: lastPart, Value: value.ToString()})

	return name, newAfter
}

func (t *Tcb) EndScope() {
	t.AddVirtPart("}\n")
	t.CurrentScope = t.CurrentScope.ParentScope
	t.CurrentScope.Parts.CloseScopePart()
}

func (t *Tcb) GetScope() *Scope {
	return t.CurrentScope
}

func (t *Tcb) GetDirectiveIdent(directive *parser.Class) string {
	for _, d := range t.Directives {
		if d.Class.Id() == directive.Id() {
			return d.Identifier
		}
	}

	return ""
}

func (t *Tcb) NewScope() *Scope {
	scope := Scope{ParentScope: t.CurrentScope, Parts: Statement{}, Variables: []*Variable{}}
	// t.CurrentScope.ChildrenScopes = append(t.CurrentScope.ChildrenScopes, &scope)
	t.CurrentScope.AddScopePart(&scope)
	t.CurrentScope = &scope

	return &scope
}

func (t *Tcb) ToString() *Statement {
	tcb := Statement{}

	tcb.AddVirtPart(t.BuildImports())
	tcb.AddStatement(t.BuildDirectiveConstructorsStatement())

	scopes := t.RootScope.ToStatement()
	tcb.AddStatement(&scopes)

	return &tcb
}

func (t *Tcb) WithScope(builder func()) {
	t.BeginScope()
	builder()
	t.EndScope()
}

func GenerateTcb(state *parser.State, template *parser.Class, root *sitter.Node, content []byte) *Statement {
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

	typeParameters := tcb.Class.Snapshot().TypeParameters
	if len(typeParameters) > 0 {
		tcb.GetScope().AddVirtPart("<")
		for i := range typeParameters {
			if i > 0 {
				tcb.GetScope().AddVirtPart(", ")
			}

			tcb.GetScope().AddVirtPart("any")
		}
		tcb.GetScope().AddVirtPart(">")
	}

	tcb.GetScope().AddVirtPart(") ")
}
