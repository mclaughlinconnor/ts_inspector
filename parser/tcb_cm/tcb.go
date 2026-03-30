package tcb_cm

import (
	"strconv"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

type Tcb struct {
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

func (t *Tcb) AddPart(part string) {
	t.GetScope().AddPart(part)
}

func (t *Tcb) BeginScope() {
	t.NewScope()
	t.AddPart("{\n")
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
