package tcb

import (
	"regexp"
	"strconv"
	"ts_inspector/parser"
)

type Identifier = int

type Context struct {
	currentId   int
	id          int
	imports     []*Import
	oobRecorder *OobRecorder
	scopes      []*Scope
	state       *parser.State
	file        *parser.File
}

type Import struct {
	foreignIdentifier string
	localIdentifier   string
	filename          string
}

type OobRecorder struct{}

func InitTcb() {
	ir, err := regexp.Compile(`[\p{L}\$_][\p{L}\d\$_]*`)
	if err != nil {
		panic(err)
	}

	identifierRegex = ir

	initTcbExpression()
}

func IdentifierName(id Identifier) string {
	return "_t" + strconv.Itoa(id)
}

func (o *OobRecorder) duplicateTemplateVar(a any, b any, c any) {}
func (o *OobRecorder) conflictingDeclaration(a any, b any)      {}

// TODO: check all usages for params
func (t *Context) allocateId(a any, b any) Identifier {
	id := t.currentId
	t.currentId += 1

	return id
}

func (t *Context) envReference(class *parser.Class) string {
	localIdentifier := class.Snapshot().Name + strconv.Itoa(t.allocateId(nil, nil))

	i := &Import{
		filename:          class.Snapshot().File.Filename(),
		foreignIdentifier: class.Snapshot().Name,
		localIdentifier:   localIdentifier,
	}
	t.imports = append(t.imports, i)

	return localIdentifier
}

type TcbExpr struct {
	TcbOp
	Source string
}

type TcbOp interface {
	CircularFallback() TcbExpr
	Execute() *Identifier
	Optional() bool
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

	statement.AddPart("var ")
	statement.AddPart(IdentifierName(id))

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

	statement.AddPart(IdentifierName(id))
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
