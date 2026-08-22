package tcb

import (
	"fmt"
	"path/filepath"
	"slices"
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

type Import struct {
	Class      *parser.Class
	File       *parser.File
	Identifier string
}

type Pipe struct {
	Class      *parser.Class
	Identifier string
	Statement  *Statement
}

type Tcb struct {
	Ast                  *Ast
	Class                *parser.Class
	CurrentScope         *Scope
	Directives           []*Directive
	Imports              []*Import
	Pipes                []*Pipe
	RootScope            *Scope
	State                *parser.State
	TagBoundaryPartStack utils.Stack[*Part]
}

func (t *Tcb) AddAssignment(identifer string, identNode *sitter.Node, value *Statement) {
	if identNode == nil {
		t.AddVirtPart(identifer)
	} else {
		t.AddRealPart(identifer, identNode)
	}

	t.AddVirtPart(" = ")
	t.AddStatement(value)
	t.AddVirtPart(";\n")
}

func (t *Tcb) AddDirectiveConstructor(identifer string, directive *parser.Class, statement *Statement, isConstructor bool) {
	constructor := Directive{Class: directive, Identifier: identifer, IsConstructor: isConstructor, Statement: statement}
	t.Directives = append(t.Directives, &constructor)
}

func (t *Tcb) AddImport(class *parser.Class) string {
	f := class.Snapshot().File
	uri := f.Snapshot().URI

	var i *Import

	index := slices.IndexFunc(t.Imports, func(i *Import) bool { return i.File.Snapshot().URI == uri })
	if index != -1 {
		i = t.Imports[index]
	} else {
		i = &Import{Class: class, File: f, Identifier: utils.GetNextStringId()}
		t.Imports = append(t.Imports, i)
	}

	return "i" + i.Identifier + "." + class.Snapshot().Name
}

func (t *Tcb) AddPipeDeclaration(identifer string, pipe *parser.Class, statement *Statement) {
	constructor := Pipe{Class: pipe, Identifier: identifer, Statement: statement}
	t.Pipes = append(t.Pipes, &constructor)
}

func (t *Tcb) AddPart(part *Part) {
	t.GetScope().AddPart(part)
}

func (t *Tcb) AddVirtPart(part string) {
	t.GetScope().AddVirtPart(part)
}

func (t *Tcb) AddStatement(parts *Statement) {
	t.GetScope().AddStatement(parts)
}

func (t *Tcb) AddStatementAfterPart(parts *Statement, after *Part) *Part {
	stack := utils.NewStack[*Scope]()
	visited := map[*Scope]bool{}

	for _, part := range t.RootScope.Parts.Parts {
		if part.scope == nil {
			continue
		}

		stack.Push(part.scope)
	}

	for !stack.IsEmpty() {
		scope := *stack.Pop()

		if visited[scope] {
			continue
		}

		visited[scope] = true

		newAfter := scope.AddStatementAfterPart(parts, after)
		if newAfter != nil {
			return newAfter
		}

		for _, part := range scope.Parts.Parts {
			if part.scope == nil {
				continue
			}

			stack.Push(part.scope)
		}
	}

	return nil
}

func (t *Tcb) AddRealPart(part string, node *sitter.Node) {
	t.GetScope().AddRealPart(part, node)
}

func (t *Tcb) BeginScope() {
	t.NewScope()
	t.AddVirtPart("{\n")
}

func (t *Tcb) BeginRealScope(node *sitter.Node) {
	t.NewScope()
	t.AddRealPart("{\n", node)
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
		relative, _ := filepath.Rel(utils.PathDir(path), ifname)
		relativePath := "./" + strings.TrimSuffix(relative, filepath.Ext(relative))

		fmt.Fprintf(&sb, "import * as i%s from '%s';\n", i.Identifier, relativePath)
	}

	return sb.String()
}

func (t *Tcb) BuildPipeConstructorsStatement() *Statement {
	statement := Statement{}

	for _, pipe := range t.Pipes {
		statement.AddStatement(pipe.Statement)
	}

	return &statement
}

func (t *Tcb) CreateVarInCurrentScope(value *Statement, alias string) string {
	return t.CreateVarInScope(value, t.GetScope(), alias)
}

func (t *Tcb) CreateVarInRootScope(value *Statement, alias string) string {
	// TODO: should probably put the var at the top of the scope, not at the end
	return t.CreateVarInScope(value, t.RootScope, alias)
}

func (t *Tcb) CreateVarInScope(value *Statement, scope *Scope, alias string) string {
	if v := t.GetScope().GetVariableByAlias(alias); v != nil {
		return v.Identifier
	}

	if v := scope.GetVariableByAlias(value.ToString()); v != nil {
		return v.Identifier
	}

	name := "_t" + utils.GetNextStringId()
	scope.AddVirtPart("var ")
	scope.AddVirtPart(name)
	scope.AddVirtPart(" = ")

	scope.AddStatement(value)

	scope.AddVirtPart(";\n")

	lastPart := t.CurrentScope.Parts.GetLastPart()

	var a string
	if alias != "" {
		a = alias
	} else {
		a = value.ToString()
	}

	scope.AddVariable(&Variable{Alias: a, Identifier: name, LastPart: lastPart, Value: value.ToString()})

	return name
}

func (t *Tcb) CreateVarAfterPart(value *Statement, alias string, after *Part) (string, *Part) {
	if v := t.GetScope().GetVariableByAlias(alias); v != nil {
		return v.Identifier, v.LastPart
	}

	if v := t.GetScope().GetVariableByAlias(value.ToString()); v != nil {
		return v.Identifier, v.LastPart
	}

	name := "_t" + utils.GetNextStringId()
	statement := Statement{}
	statement.AddVirtPart("var ")
	statement.AddVirtPart(name)
	statement.AddVirtPart(" = ")

	statement.AddStatement(value)

	statement.AddVirtPart(";\n")

	newAfter := t.AddStatementAfterPart(&statement, after)

	lastPart := t.CurrentScope.Parts.GetLastPart()

	var a string
	if alias != "" {
		a = alias
	} else {
		a = value.ToString()
	}

	t.GetScope().AddVariable(&Variable{Alias: a, Identifier: name, LastPart: lastPart, Value: value.ToString()})

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

func (t *Tcb) GetPipeIdent(pipe *parser.Class) string {
	for _, p := range t.Pipes {
		if p.Class.Id() == pipe.Id() {
			return p.Identifier
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
	tcb.AddStatement(t.BuildPipeConstructorsStatement())

	scopes := t.RootScope.ToStatement()
	tcb.AddStatement(&scopes)

	return &tcb
}

func GenerateTcb(state *parser.State, template *parser.Class, root *sitter.Node, content []byte) (*Statement, error) {
	scope := &Scope{}

	tcb := Tcb{
		CurrentScope: scope,
		RootScope:    scope,
		State:        state,
		Class:        template,
	}

	ast, err := Parse(root, content, &tcb)
	if err != nil {
		return nil, err
	}

	buildTemplatePreamble(&tcb)

	tcb.BeginScope()
	err = ast.Render()
	if err != nil {
		return nil, err
	}
	tcb.EndScope()

	return tcb.ToString(), nil
}

func InitTcb() {
	initTcbExpression()
	initAstParser()
}

func buildTemplatePreamble(tcb *Tcb) {
	tcb.GetScope().AddVirtPart("function _tcb")
	tcb.GetScope().AddVirtPart(utils.GetNextStringId())
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
