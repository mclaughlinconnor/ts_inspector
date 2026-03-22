package tcb

import (
	"regexp"
	"strconv"
	"ts_inspector/parser"
	"ts_inspector/parser/ast"
)

type Expression = []string
type Identifier = int

type Import struct {
	foreignIdentifier string
	localIdentifier   string
	filename          string
}

type Scope struct {
	parentScope *Scope
	statements  []Statement
	tcb         *Context
}

type Statement struct {
	parts []string
}

type Context struct {
	currentId int
	imports   []*Import
	scopes    []*Scope
}

func InitTcb() {
	ir, err := regexp.Compile(`[\p{L}\$_][\p{L}\d\$_]*`)
	if err != nil {
		panic(err)
	}

	identifierRegex = ir

	initTcbExpression()
}

func (s *Scope) addStatement(statement Statement) {
	s.statements = append(s.statements, statement)
}

func (s *Scope) render() []Statement {
	line := []string{"__STATEMENT__;"}
	first := Statement{line}

	return []Statement{first}
}

func (s *Scope) resolve(ident *ast.Node, directive *ast.Node) string {
	return "__RESOLVED__"
}

func (s *Statement) AddCodeBlock(builder func()) {
	s.parts = append(s.parts, "{\n")
	builder()
	s.parts = append(s.parts, "\n}")
}

func (s *Statement) AddPart(part string) {
	s.parts = append(s.parts, part)
}

func (s *Statement) AppendStatement(statement Statement) {
	s.parts = append(s.parts, statement.parts...)
	s.parts = append(s.parts, "\n")
}

func (t *Context) allocateId() int {
	id := t.currentId

	t.currentId += 1

	return id
}

func (t *Context) envReference(class *parser.Class) string {
	localIdentifier := class.Snapshot().Name + strconv.Itoa(t.allocateId())

	i := &Import{
		filename:          class.Snapshot().File.Filename(),
		foreignIdentifier: class.Snapshot().Name,
		localIdentifier:   localIdentifier,
	}
	t.imports = append(t.imports, i)

	return localIdentifier
}

// Scope.forNodes
func scopeForNodes(tcb *Context, parentScope *Scope, scopedNode *ast.Node, children []*ast.Node, guard *Expression) Scope {
	scope := Scope{parentScope, []Statement{}, tcb}

	return scope
}

type TcbExpr struct {
	Source string
}

type TcbOp interface {
	Optional() bool
	Execute() *TcbExpr
	CircularFallback() TcbExpr
}

/**
 * Create a `ts.VariableStatement` which declares a variable without explicit initialization.
 *
 * The initializer `null!` is used to bypass strict variable initialization checks.
 *
 * Unlike with `tsCreateVariable`, the type of the variable is explicitly specified.
 */
func tsDeclareVariable(id Identifier, ttype Expression, initializer *Expression) Statement {
	// When we create a variable like `var _t1: boolean = null!`, TypeScript actually infers `_t1`
	// to be `never`, instead of a `boolean`. To work around it, we cast the value
	// in the initializer, e.g. `var _t1 = null! as boolean;`.

	statement := Statement{}

	statement.AddPart("var")
	statement.AddPart(strconv.Itoa(id))

	if initializer != nil {
		statement.AddPart(" : ")
		statement.AddPart(" = ")

		for _, i := range *initializer {
			statement.AddPart(i)
		}

		statement.AddPart(";")
	} else {
		statement.AddPart(" = null! as ")

		for _, t := range ttype {
			statement.AddPart(t)
		}

		statement.AddPart(";")
	}

	return statement
}

/**
 * Create a `ts.VariableStatement` that initializes a variable with a given expression.
 *
 * Unlike with `tsDeclareVariable`, the type of the variable is inferred from the initializer
 * expression.
 */
func tsCreateVariable(id Identifier, initializer Expression, isConst bool) Statement {
	statement := Statement{}
	if isConst {
		statement.AddPart("const ")
	} else {
		statement.AddPart("var ")
	}

	statement.AddPart(strconv.Itoa(id))
	statement.AddPart(" = ")

	for _, i := range initializer {
		statement.AddPart(i)
	}

	statement.AddPart(";")

	return statement
}

var identifierRegex *regexp.Regexp

func isJavascriptIdentifier(identifer string) bool {
	return identifierRegex.MatchString(identifer)
}
