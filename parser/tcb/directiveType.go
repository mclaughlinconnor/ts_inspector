package tcb

import (
	"strings"
	"ts_inspector/parser"
	"ts_inspector/parser/ast"
)

/**
 * A `TcbOp` which constructs an instance of a non-generic directive _without_ setting any of its
 * inputs. Inputs are later set in the `TcbDirectiveInputsOp`. Type checking was found to be
 * faster when done in this way as opposed to `TcbDirectiveCtorOp` which is only necessary when the
 * directive is generic.
 *
 * Executing this operation returns a reference to the directive instance variable with its inferred
 * type.
 */
func handleTcbNonGenericDirectiveType(state *parser.State, file *parser.File, tcb *Context, scope *Scope, node ast.Node, dir ast.Node) {
	/**
	 * Creates a variable declaration for this op's directive of the argument type. Returns the id of
	 * the newly created variable.
	 */
	directiveTypeBase(state, file, tcb, scope, node, dir)
}

/**
 * A `TcbOp` which constructs an instance of a generic directive with its generic parameters set
 * to `any` type. This op is like `TcbDirectiveTypeOp`, except that generic parameters are set to
 * `any` type. This is used for situations where we want to avoid inlining.
 *
 * Executing this operation returns a reference to the directive instance variable with its generic
 * type parameters set to `any`.
 */
func handleGenericDirectiveTypeWithAnyParams(state *parser.State, file *parser.File, tcb *Context, scope *Scope, node ast.Node, dir ast.Node) {
	directiveTypeBase(state, file, tcb, scope, node, dir)
}

/**
 * A `TcbOp` which constructs an instance of a directive. For generic directives, generic
 * parameters are set to `any` type.
 */
func directiveTypeBase(state *parser.State, file *parser.File, tcb *Context, scope *Scope, node ast.Node, dir ast.Node) Identifier {
	exitAny := func() int {
		id := tcb.allocateId()
		scope.addStatement(tsDeclareVariable(id, Expression{"any"}, nil))
		return id
	}

	fileClass := file.Snapshot().Classes[0]
	if fileClass == nil {
		return exitAny()
	}

	cls := dir.Tag.ResolveSourceClassOfTag(state, file.Snapshot().Classes[0])
	if cls == nil {
		return exitAny()
	}

	clsId := tcb.envReference(cls)

	ttype := Expression{}
	ttype = append(ttype, clsId)

	typeParameters := cls.Snapshot().TypeParameters
	if len(typeParameters) > 0 {
		ttype = append(ttype, "<")

		aany := []string{}
		for _, _ = range typeParameters {
			aany = append(aany, "any")
		}

		ttype = append(ttype, strings.Join(aany, ", "))

		ttype = append(ttype, ">")
	}

	id := tcb.allocateId()

	scope.addStatement(tsDeclareVariable(id, ttype, nil))

	return id
}
