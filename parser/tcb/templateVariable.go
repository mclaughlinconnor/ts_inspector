package tcb

import (
	"strings"
)

type TcbTemplateVariableOp struct {
	TcbOp
	tcb      *Context
	scope    *Scope
	template *TmplAstNode
	variable *TmplAstVariable
}

func (o TcbTemplateVariableOp) Optional() bool { return false }
func (o TcbTemplateVariableOp) Execute() Identifier {
	// Look for a context variable for the template.
	ctx := o.scope.resolve(o.template, nil)
	// Allocate an identifier for the TmplAstVariable, and initialize it to a read of the variable
	// on the template context.
	id := o.tcb.allocateId(nil, nil)
	statement := Statement{}

	statement.AddPart("var ")
	statement.AddPart(IdentifierName(id))
	statement.AddPart(" = ")
	statement.AddPart(IdentifierName(ctx))

	name := o.variable.Name
	if name == "" {
		name = "$IMPLICIT"
	}

	if isJavascriptIdentifier(name) {
		statement.AddPart(".")
		statement.AddPart(name)
	} else {
		statement.AddPart("[\"")
		statement.AddPart(strings.ReplaceAll(name, "\"", "\\\""))
		statement.AddPart("\"]")
	}

	statement.AddPart(";")

	o.scope.AddStatement(statement)

	return id
}

func (o TcbTemplateVariableOp) CircularFallback() TcbExpr { return TcbExpr{Source: "null!"} }

/**
 * A `TcbOp` which creates an expression for particular let- `TmplAstVariable` on a
 * `TmplAstTemplate`'s context.
 *
 * Executing this operation returns a reference to the variable variable (lol).
 */
func handleTemplateVariable(tcb *Context, scope *Scope, variable *TmplAstNode, template *TmplAstNode) Identifier {
	op := &TcbTemplateVariableOp{tcb: tcb, scope: scope, template: template, variable: variable.Variable}

	return op.Execute()
}
