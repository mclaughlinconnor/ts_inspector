package tcb

import (
	"strconv"
	"strings"
	"ts_inspector/parser/ast"
)

/**
 * A `TcbOp` which creates an expression for particular let- `TmplAstVariable` on a
 * `TmplAstTemplate`'s context.
 *
 * Executing this operation returns a reference to the variable variable (lol).
 */
func handleTemplateVariable(tcb *Context, scope *Scope, variable *ast.Node, template *ast.Node) Identifier {
	// Look for a context variable for the template.
	ctx := scope.resolve(template, nil)

	// Allocate an identifier for the TmplAstVariable, and initialize it to a read of the variable
	// on the template context.
	id := tcb.allocateId()
	statement := Statement{}

	statement.AddPart("var ")
	statement.AddPart(strconv.Itoa(id))
	statement.AddPart(" = ")
	statement.AddPart(ctx)

	var name string
	if variable.Variable.Name == "" {
		name = "$IMPLICIT"
	} else {
		name = variable.Variable.Name
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

	scope.addStatement(statement)

	return id
}
