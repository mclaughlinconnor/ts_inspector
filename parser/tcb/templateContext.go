package tcb

/**
 * A `TcbOp` which generates a variable for a `TmplAstTemplate`'s context.
 *
 * Executing this operation returns a reference to the template's context variable.
 */
func handleTemplateContext(tcb *Context, scope *Scope) Identifier {
	// Allocate a template ctx variable and declare it with an 'any' type. The type of this variable
	// may be narrowed as a result of template guard conditions.
	ctx := tcb.allocateId()

	statement := Statement{}
	statement.AddPart("var ${ctx} = null! as any;")

	scope.addStatement(statement)

	return ctx
}
