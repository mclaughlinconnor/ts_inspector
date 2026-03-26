package tcb

type TcbTemplateContextOp struct {
	tcb   *Context
	scope *Scope
}

func (o TcbTemplateContextOp) Optional() bool { return true }
func (o TcbTemplateContextOp) Execute() *Identifier {
	id := o.tcb.allocateId(nil, nil)

	statement := Statement{}
	statement.AddPart("var ")
	statement.AddPart(IdentifierName(id))
	statement.AddPart(" = null! as any;")

	o.scope.addStatementStatement(statement)

	return &id
}
func (o TcbTemplateContextOp) CircularFallback() TcbExpr { return TcbExpr{Source: "null!"} }

/**
 * A `TcbOp` which generates a variable for a `TmplAstTemplate`'s context.
 *
 * Executing this operation returns a reference to the template's context variable.
 */
func handleTemplateContext(tcb *Context, scope *Scope) *Identifier {
	op := &TcbTemplateContextOp{tcb: tcb, scope: scope}
	return op.Execute()
}
