package tcb

import (
	"ts_inspector/parser/ast"
)

/**
 * A `TcbOp` which creates an expression for a native DOM element (or web component) from a
 * `TmplAstElement`.
 *
 * Executing this operation returns a reference to the element variable.
 */
func handleElement(tcb *Context, scope *Scope, tag *ast.Node) Identifier {
	id := tcb.allocateId()

	variable := tsCreateVariable(id, Expression{"document.createElement(\"${element.name}\")"}, false)
	scope.addStatement(variable)

	return id
}
