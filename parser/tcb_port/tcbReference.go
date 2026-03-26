package tcb_port

import (
	"ts_inspector/parser"
)

/**
 * A `TcbOp` which creates a variable for a local ref in a template.
 * The initializer for the variable is the variable expression for the directive, template, or
 * element the ref refers to. When the reference is used in the template, those TCB statements will
 * access this variable as well. For example:
 * ```
 * var _t1 = document.createElement('div');
 * var _t2 = _t1;
 * _t2.value
 * ```
 * This operation supports more fluent lookups for the `TemplateTypeChecker` when getting a symbol
 * for a reference. In most cases, this isn't essential; that is, the information for the symbol
 * could be gathered without this operation using the `BoundTarget`. However, for the case of
 * ng-template references, we will need this reference variable to not only provide a location in
 * the shim file, but also to narrow the variable to the correct `TemplateRef<T>` type rather than
 * `TemplateRef<any>` (this work is still TODO).
 *
 * Executing this operation returns a reference to the directive instance variable with its inferred
 * type.
 */
func handleReference(state *parser.State, file *parser.File, tcb *Context, scope *Scope, node TmplAstNode, host TmplAstNode, target TmplAstNode) Identifier {
	id := tcb.allocateId(nil, nil)

	//   val reference = Expression {
	//     append(if (target is TmplAstDirectiveContainer)
	//              scope.resolve(target)
	//            else
	//              scope.resolve(host, target as TmplDirectiveMetadata),
	//            node.valueSpan, supportTypes = true)
	//   }
	//
	//   // The reference is either to an element, an <ng-template> node, or to a directive on an
	//   // element or template.
	//   val initializer = Expression {
	//     if ((target is TmplAstElement && !tcb.env.config.checkTypeOfDomReferences) ||
	//         !tcb.env.config.checkTypeOfNonDomReferences) {
	//       // References to DOM nodes are pinned to 'any' when `checkTypeOfDomReferences` is `false`.
	//       // References to `TemplateRef`s and directives are pinned to 'any' when
	//       // `checkTypeOfNonDomReferences` is `false`.
	//       append(reference).append(" as any")
	//     }
	//     else if (target is TmplAstTemplate) {
	//       // Direct references to an <ng-template> node simply require a value of type
	//       // `TemplateRef<any>`. To get this, an expression of the form
	//       // `(_t1 as any as TemplateRef<any>)` is constructed.
	//       append("(")
	//       append(reference).append(" as any as ")
	//       append(tcb.env.referenceExternalType(ANGULAR_CORE_PACKAGE, TEMPLATE_REF))
	//       append("<any>)")
	//     }
	//     else {
	//       append(reference)
	//     }
	//   }
	//   scope.addStatement(tsCreateVariable(id, initializer))
	//   return id
	// }

	return id
}
