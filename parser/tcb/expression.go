package tcb

/**
 * A `TcbOp` which renders an Angular expression (e.g. `{{foo() && bar.baz}}`).
 *
 * Executing this operation returns nothing.
 */
func handleExpression(tcb *Context, scope *Scope, expression *Expression, isBoundText bool) *Identifier {
	if expression != nil {
		expr := AstToTypescript(expression)

		statement := Statement{}
		if isBoundText {
			statement.AddPart("\"\" + (")
			statement.AddPart(expr)
			statement.AddPart(");")
		} else {
			statement.AddPart("(")
			statement.AddPart(expr)
			statement.AddPart(");")
		}
	}

	return nil
}
