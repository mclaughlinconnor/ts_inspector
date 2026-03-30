package tcb_cm

import (
	"slices"
	"strconv"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type Import struct {
	Class      *parser.Class
	File       *parser.File
	Identifier string
}

type Tcb struct {
	Imports  []*Import
	NextId   int
	Scope    *utils.Stack[Scope]
	State    *parser.State
	Template *parser.Class
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

func (t *Tcb) AddAssignment(identifer string, value StatementParts) {
	t.AddPart(identifer)
	t.AddPart(" = ")

	for _, p := range value.Parts {
		t.AddPart(p)
	}

	t.AddPart(";\n")
}

func (t *Tcb) AddImport(class *parser.Class) string {
	f := class.Snapshot().File
	uri := f.Snapshot().URI

	var i *Import

	index := slices.IndexFunc(t.Imports, func(i *Import) bool { return i.File.Snapshot().URI == uri })
	if index != -1 {
		i = t.Imports[index]
	} else {
		i = &Import{Class: class, Identifier: t.GetNextIdString()}
		t.Imports = append(t.Imports, i)
	}

	return "i" + i.Identifier + "." + class.Snapshot().Name
}

func (t *Tcb) AddPart(part string) {
	t.GetScope().AddPart(part)
}

func (t *Tcb) BeginScope() {
	t.NewScope()
	t.AddPart("{\n")
}

func (t *Tcb) CreateVar(value StatementParts) string {
	if v := t.GetScope().GetVariable(value); v != nil {
		return v.Identifier
	}

	name := "_t" + t.GetNextIdString()
	t.AddPart("var ")
	t.AddPart(name)
	t.AddPart(" = ")

	for _, p := range value.Parts {
		t.AddPart(p)
	}

	t.AddPart(";\n")

	t.GetScope().AddVariable(&Variable{Identifier: name, Value: value.ToString()})

	return name
}

func (t *Tcb) EndScope() {
	t.AddPart("\n}")
	t.Scope.Pop()
}

func (t *Tcb) GetScope() *Scope {
	return t.Scope.Peek()
}

func (t *Tcb) NewScope() {
	scope := Scope{ParentScope: t.Scope.Peek(), Parts: StatementParts{}, Variables: []*any{}}
	t.Scope.Push(scope)
}

func (t *Tcb) WithScope(builder func()) {
	t.BeginScope()
	builder()
	t.EndScope()
}

func GenerateTcb(state *parser.State, template *parser.Class, ast *Ast) string {
	scopeStack := utils.NewStack[Scope]()
	tcb := Tcb{NextId: 0, Scope: scopeStack, State: state, Template: template}

	buildTemplatePreamble(&tcb)

	tcb.WithScope(func() {
		ast.Render()
	})

	return ""
}

func buildTemplatePreamble(tcb *Tcb) {
	tcb.GetScope().AddPart("function _tcb")
	tcb.GetScope().AddPart(tcb.GetNextIdString())
	tcb.GetScope().AddPart("(this: ")
	// tcb.GetScope().AddPart() // i0
	tcb.GetScope().AddPart("." + tcb.Template.Snapshot().Name)
	tcb.GetScope().AddPart(") ")
}
