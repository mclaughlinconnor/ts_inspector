package tcb

type TcbElementOp struct {
	tcb     *Context
	scope   *Scope
	element *Node
}

func (o *TcbElementOp) Optional() bool { return true }
func (o *TcbElementOp) Execute() Identifier {
	id := o.tcb.allocateId(nil, nil)
	o.scope.addStatementStatement(tsCreateVariable(id, []string{"document.createElement(\"", o.element.Tag.Name, "\")"}, true))
	return id
}
func (o *TcbElementOp) CircularFallback() TcbExpr { return TcbExpr{Source: "null!"} }

/**
 * A `TcbOp` which creates an expression for a native DOM element (or web component) from a
 * `TmplAstElement`.
 *
 * Executing this operation returns a reference to the element variable.
 */
func handleElement(tcb *Context, scope *Scope, tag *Node) Identifier {
	op := &TcbElementOp{tcb: tcb, scope: scope, element: tag}
	return op.Execute()
}
