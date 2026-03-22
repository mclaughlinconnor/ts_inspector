package tcb

import (
	"strings"
	"ts_inspector/parser/ast"
)

/**
 * Local scope within the type check block for a particular template.
 *
 * The top-level template and each nested `<ng-template>` have their own `Scope`, which exist in a
 * hierarchy. The structure of this hierarchy mirrors the syntactic scopes in the generated type
 * check block, where each nested template is encased in an `if` structure.
 *
 * As a template's `TcbOp`s are executed in a given `Scope`, statements are added via
 * `addStatement()`. When this processing is complete, the `Scope` can be turned into a `ts.Block`
 * via `renderToBlock()`.
 *
 * If a `TcbOp` requires the output of another, it can call `resolve()`.
 */
type Scope struct {
	tcb    *Context
	parent *Scope
	guard  *Expression

	/**
	* A queue of operations which need to be performed to generate the TCB code for this scope.
	*
	* This array can contain either a `TcbOp` which has yet to be executed, or a `Expression|null`
	* representing the memoized result of executing the operation. As operations are executed, their
	* results are written into the `opQueue`, overwriting the original operation.
	*
	* If an operation is in the process of being executed, it is temporarily overwritten here with
	* `INFER_TYPE_FOR_CIRCULAR_OP_EXPR`. This way, if a cycle is encountered where an operation
	* depends transitively on its own result, the inner operation will infer the least narrow type
	* that fits instead. This has the same semantics as TypeScript itself when types are referenced
	* circularly.
	 */
	opQueue []*ast.Node

	/**
	* A map of `TmplAstElement`s to the index of their `TcbElementOp` in the `opQueue`
	 */
	elementOpMap map[*ast.Node]int

	/**
	* A map of maps which tracks the index of `TcbDirectiveCtorOp`s in the `opQueue` for each
	* directive on a `TmplAstElement` or `TmplAstTemplate` node.
	 */
	directiveOpMap map[*ast.Node]map[*ast.Node]int

	/**
	* A map of `TmplAstReference`s to the index of their `TcbReferenceOp` in the `opQueue`
	 */
	referenceOpMap map[*ast.Node]int

	/**
	* Map of immediately nested <ng-template>s (within this `Scope`) represented by `TmplAstTemplate`
	* nodes to the index of their `TcbTemplateContextOp`s in the `opQueue`.
	 */
	templateCtxOpMap map[*ast.Node]int

	/**
	* Map of variables declared on the template that created this `Scope` (represented by
	* `TmplAstVariable` nodes) to the index of their `TcbVariableOp`s in the `opQueue`, or to
	* pre-resolved variable identifiers.
	 */
	varMap map[*ast.Node]Identifier

	/**
	* A map of the names of `TmplAstLetDeclaration`s to the index of their op in the `opQueue`.
	*
	* Assumes that there won't be duplicated `@let` declarations within the same scope.
	 */
	letDeclOpMap map[*ast.Node]LetDeclOpMapRecord

	/**
	* Statements for this template.
	*
	* Executing the `TcbOp`s in the `opQueue` populates this array.
	 */
	statements []Statement
}

/**
* Names of the for loop context variables and their types.
 */
var forLoopContextVariableTypes = map[string]string{
	"$index": "number",
	"$first": "boolean",
	"$last":  "boolean",
	"$even":  "boolean",
	"$odd":   "boolean",
	"$count": "number",
}

/**
* Constructs a `Scope` given either a `TmplAstTemplate` or a list of `TmplAstNode`s.
*
* @param tcb the overall context of TCB generation.
* @param parentScope the `Scope` of the parent template (if any) or `null` if this is the root
* `Scope`.
* @param scopedNode Node that provides the scope around the child nodes (e.g. a
* `TmplAstTemplate` node exposing variables to its children).
* @param children Child nodes that should be appended to the TCB.
* @param guard an expression that is applied to this scope for type narrowing purposes.
 */
func scopeForNodes(tcb *Context, parentScope *Scope, scopedNode *ast.Node, children []*ast.Node, guard *Expression) Scope {
	var guardExpr *Expression = nil

	if guard != nil {
		guardExpr := Expression{}
		guardExpr = append(guardExpr, "(")
		guardExpr = append(guardExpr, strings.Join(guard, ""))
		guardExpr = append(guardExpr, ")")
	}

	scope := Scope{tcb: tcb, parent: parentScope, guard: guardExpr}

	// If given an actual `TmplAstTemplate` instance, then process any additional information it has.
	if scopedNode.Kind == ast.KindTmplAstTemplate {
		// The template"s variable declarations need to be added as `TcbVariableOp`s.
		varMap := map[string]*ast.TmplAstVariable{}

		for v := range scopedNode.variables.values {
			// Validate that variables on the `TmplAstTemplate` are only declared once.
			if !slices.Contains(varMap, v.name) {
				varMap[v.name] = v
			} else {
				firstDecl := varMap[v.name]
				tcb.oobRecorder.duplicateTemplateVar(tcb.id, v, firstDecl)
			}

			scope.registerVariable(v, TcbTemplateVariableOp(tcb, scope, scopedNode, v))
		}

	}

	//
	//     if (scopedNode is TmplAstTemplate) {
	//       // The template"s variable declarations need to be added as `TcbVariableOp`s.
	//       val varMap = mutableMapOf<String, TmplAstVariable>()
	//
	//       for (v in scopedNode.variables.values) {
	//         // Validate that variables on the `TmplAstTemplate` are only declared once.
	//         if (!varMap.contains(v.name)) {
	//           varMap[v.name] = v
	//         }
	//         else {
	//           val firstDecl = varMap[v.name]!!
	//           tcb.oobRecorder.duplicateTemplateVar(tcb.id, v, firstDecl)
	//         }
	//         scope.registerVariable(v, TcbTemplateVariableOp(tcb, scope, scopedNode, v))
	//       }
	//     }
	//     else if (scopedNode is TmplAstIfBlockBranch) {
	//       val expression = scopedNode.expression
	//       val expressionAlias = scopedNode.expressionAlias
	//       if (expression != null && expressionAlias != null) {
	//         scope.registerVariable(expressionAlias,
	//                                TcbBlockVariableOp(
	//                                  tcb, scope, tcbExpression(expression, tcb, scope), expressionAlias))
	//       }
	//     }
	//     else if (scopedNode is TmplAstForLoopBlock) {
	//       // Register the variable for the loop so it can be resolved by
	//       // children. It'll be declared once the loop is created.
	//       scopedNode.item?.let {
	//         val loopInitializer = tcb.allocateId(it)
	//         scope.varMap[it] = loopInitializer
	//       }
	//
	//       for ((name, variables) in scopedNode.contextVariables.entrySet()) {
	//         val typeName = forLoopContextVariableTypes[name] ?: "any"
	//         for ((variable, initializer) in variables) {
	//           scope.registerVariable(variable, TcbBlockImplicitVariableOp(tcb, scope, Expression(typeName),
	//                                                                       variable, initializer?.let { tcbExpression(it, tcb, scope) }))
	//         }
	//       }
	//     }
	//     for (node in children) {
	//       scope.appendNode(node)
	//     }
	//     // Once everything is registered, we need to check if there are `@let`
	//     // declarations that conflict with other local symbols defined after them.
	//     for (variable in scope.varMap.keys) {
	//       scope.checkConflictingLet(variable)
	//     }
	//     for (ref in scope.referenceOpMap.keys) {
	//       scope.checkConflictingLet(ref)
	//     }
	//     return scope
}

//     val scope = Scope(tcb, parentScope, guard?.let { Expression { append("(").append(guard).append(")") } })
//
//     // If given an actual `TmplAstTemplate` instance, then process any additional information it
//     // has.
//     if (scopedNode is TmplAstTemplate) {
//       // The template"s variable declarations need to be added as `TcbVariableOp`s.
//       val varMap = mutableMapOf<String, TmplAstVariable>()
//
//       for (v in scopedNode.variables.values) {
//         // Validate that variables on the `TmplAstTemplate` are only declared once.
//         if (!varMap.contains(v.name)) {
//           varMap[v.name] = v
//         }
//         else {
//           val firstDecl = varMap[v.name]!!
//           tcb.oobRecorder.duplicateTemplateVar(tcb.id, v, firstDecl)
//         }
//         scope.registerVariable(v, TcbTemplateVariableOp(tcb, scope, scopedNode, v))
//       }
//     }
//     else if (scopedNode is TmplAstIfBlockBranch) {
//       val expression = scopedNode.expression
//       val expressionAlias = scopedNode.expressionAlias
//       if (expression != null && expressionAlias != null) {
//         scope.registerVariable(expressionAlias,
//                                TcbBlockVariableOp(
//                                  tcb, scope, tcbExpression(expression, tcb, scope), expressionAlias))
//       }
//     }
//     else if (scopedNode is TmplAstForLoopBlock) {
//       // Register the variable for the loop so it can be resolved by
//       // children. It'll be declared once the loop is created.
//       scopedNode.item?.let {
//         val loopInitializer = tcb.allocateId(it)
//         scope.varMap[it] = loopInitializer
//       }
//
//       for ((name, variables) in scopedNode.contextVariables.entrySet()) {
//         val typeName = forLoopContextVariableTypes[name] ?: "any"
//         for ((variable, initializer) in variables) {
//           scope.registerVariable(variable, TcbBlockImplicitVariableOp(tcb, scope, Expression(typeName),
//                                                                       variable, initializer?.let { tcbExpression(it, tcb, scope) }))
//         }
//       }
//     }
//     for (node in children) {
//       scope.appendNode(node)
//     }
//     // Once everything is registered, we need to check if there are `@let`
//     // declarations that conflict with other local symbols defined after them.
//     for (variable in scope.varMap.keys) {
//       scope.checkConflictingLet(variable)
//     }
//     for (ref in scope.referenceOpMap.keys) {
//       scope.checkConflictingLet(ref)
//     }
//     return scope
//   }
//
//   @JvmStatic
//   internal fun forDynamicBindings(
//     tcb: Context, dynamicBindings: List<DynamicDirectiveBinding>,
//   ): Scope {
//     val scope = Scope(tcb, null, null)
//     dynamicBindings.groupBy { it.directive }.forEach { (directive, bindings) ->
//       val directivesMetadata = directive?.let { buildMetadata(it) } ?: emptyList()
//
//       val node: `TmplAstElement|TmplAstTemplate` = TmplAstElement(
//         "div", null, directivesMetadata.toSet(),
//         bindings.filter { it.kind == INPUT_BINDING_FUN || it.kind == TWO_WAY_BINDING_FUN }
//           .associateBy({ it.name.value?.toString() ?: "<null>" }, {
//             TmplAstBoundAttribute(
//               name = it.name.value?.toString() ?: "<null>",
//               keySpan = it.name.textRange.let { TextRange(it.startOffset + 1, it.endOffset - 1) },
//               type = if (it.kind == TWO_WAY_BINDING_FUN) BindingType.TwoWay else BindingType.Property,
//               value = it.value as? JSExpression,
//               valueMappingOffset = 0,
//               sourceSpan = it.name.textRange
//             ).also {
//               tcb.markAttributeExpressionAsTranspiled(it)
//             }
//           }),
//         bindings.filter { it.kind == OUTPUT_BINDING_FUN }
//           .associateBy({ it.name.value?.toString() ?: "<null>" }, {
//             TmplAstBoundEvent(
//               name = it.name.value?.toString() ?: "<null>",
//               keySpan = it.name.textRange.let { TextRange(it.startOffset + 1, it.endOffset - 1) },
//               type = ParsedEventType.Regular,
//               handler = listOf(it.value),
//               handlerMappingOffset = 0,
//               target = null,
//               phase = null,
//               sourceSpan = it.name.textRange,
//               jsType = null,
//               fromHostBinding = false,
//             )
//           }),
//         emptyMap(), emptyMap(), null, emptyList()
//       )
//
//       val dirMap = mutableMapOf<TmplDirectiveMetadata, Int>()
//       for (directiveMetadata in directivesMetadata) {
//         var directiveOp: TcbOp
//
//         if (!directiveMetadata.isGeneric) {
//           directiveOp = TcbNonGenericDirectiveTypeOp(scope.tcb, scope, node, directiveMetadata)
//         }
//         else if (!requiresInlineTypeCtor(directiveMetadata.typeScriptClass, scope.tcb.env) ||
//                  scope.tcb.env.config.useInlineTypeConstructors) {
//           directiveOp = TcbDirectiveCtorOp(scope.tcb, scope, node, directiveMetadata)
//         }
//         else {
//           directiveOp = TcbGenericDirectiveTypeWithAnyParamsOp(scope.tcb, scope, node, directiveMetadata)
//         }
//
//         scope.opQueue.add(directiveOp)
//         dirMap[directiveMetadata] = scope.opQueue.lastIndex
//
//         scope.opQueue.add(TcbDirectiveInputsOp(tcb, scope, node, directiveMetadata, isDynamicDirective = true))
//         scope.opQueue.add(TcbDynamicDirectiveOutputsOp(tcb, scope, node, directiveMetadata))
//       }
//       scope.directiveOpMap[node] = dirMap
//     }
//     return scope
//   }
//
//
// /** Registers a local variable with a scope. */
// private fun registerVariable(variable: TmplAstVariable, op: TcbOp) {
//   this.opQueue.add(op)
//   val opIndex = this.opQueue.size - 1
//   this.varMap[variable] = opIndex
// }
//
// /**
//  * Look up a `Expression` representing the value of some operation in the current `Scope`,
//  * including any parent scope(s). This method always returns a mutable clone of the
//  * `Expression` with the comments cleared.
//  *
//  * @param node a `TmplAstNode` of the operation in question. The lookup performed will depend on
//  * the type of this node:
//  *
//  * Assuming `directive` is not present, then `resolve` will return:
//  *
//  * * `TmplAstElement` - retrieve the expression for the element DOM node
//  * * `TmplAstTemplate` - retrieve the template context variable
//  * * `TmplAstVariable` - retrieve a template let- variable
//  * * `TmplAstReference` - retrieve variable created for the local ref
//  *
//  * @param directive if present, a directive type on a `TmplAstElement` or `TmplAstTemplate` to
//  * look up instead of the default for an element or template node.
//  */
// fun resolve(
//   node: LocalSymbol,
//   directive: TmplDirectiveMetadata? = null,
// ): Identifier {
//   // Attempt to resolve the operation locally.
//   val res = this.resolveLocal(node, directive)
//   if (res != null) {
//     return res
//   }
//   else if (this.parent != null) {
//     // Check with the parent.
//     return this.parent.resolve(node, directive)
//   }
//   else {
//     throw Error("Could not resolve ${node} / ${directive}")
//   }
// }
//
// /**
//  * Add a statement to this scope.
//  */
// fun addStatement(stmt: Statement) {
//   this.statements.add(stmt)
// }
//
// fun addStatement(expr: Expression) {
//   this.statements.add(Statement { append(expr).append(";") })
// }
//
// fun addStatement(builder: Expression.ExpressionBuilder.() -> Unit) {
//   addStatement(Statement(builder))
// }
//
// /**
//  * Get the statements.
//  */
// fun render(): List<Statement> {
//   for (i in opQueue.indices) {
//     // Optional statements cannot be skipped when we are generating the TCB for use
//     // by the TemplateTypeChecker.
//     val skipOptional = !this.tcb.env.config.enableTemplateTypeChecker
//     this.executeOp(i, skipOptional)
//   }
//   return this.statements
// }
//
// /**
//  * Returns an expression of all template guards that apply to this scope, including those of
//  * parent scopes. If no guards have been applied, null is returned.
//  */
// fun guards(): Expression? {
//   var parentGuards: Expression? = null
//   if (this.parent != null) {
//     // Start with the guards from the parent scope, if present.
//     parentGuards = this.parent.guards()
//   }
//
//   if (this.guard == null) {
//     // This scope does not have a guard, so return the parent's guards as is.
//     return parentGuards
//   }
//   else if (parentGuards == null) {
//     // There's no guards from the parent scope, so this scope's guard represents all available
//     // guards.
//     return this.guard
//   }
//   else {
//     // Both the parent scope and this scope provide a guard, so create a combination of the two.
//     // It is important that the parent guard is used as left operand, given that it may provide
//     // narrowing that is required for this scope's guard to be valid.
//     return Expression {
//       append(parentGuards)
//       append(" && ")
//       append(this@Scope.guard)
//     }
//   }
// }
//
// /** Returns whether a template symbol is defined locally within the current scope. */
// fun isLocal(node: TemplateEntity): Boolean {
//   if (node is TmplAstVariable) {
//     return this.varMap.containsKey(node)
//   }
//   if (node is TmplAstLetDeclaration) {
//     return this.letDeclOpMap.containsKey(node.name)
//   }
//   return this.referenceOpMap.containsKey(node)
// }
//
// private fun resolveLocal(
//   ref: LocalSymbol,
//   directive: TmplDirectiveMetadata? = null,
// ): Identifier? {
//   if (ref is TmplAstReference && this.referenceOpMap.contains(ref)) {
//     return this.resolveOp(this.referenceOpMap[ref]!!)
//   }
//   else if (ref is TmplAstLetDeclaration && this.letDeclOpMap.containsKey(ref.name)) {
//     return this.resolveOp(this.letDeclOpMap[ref.name]!!.opIndex)
//   }
//   else if (ref is TmplAstVariable && this.varMap.contains(ref)) {
//     // Resolving a context variable for this template.
//     // Execute the `TcbVariableOp` associated with the `TmplAstVariable`.
//     val opIndexOrNode = this.varMap[ref]!!
//     return if (opIndexOrNode is Int) this.resolveOp(opIndexOrNode) else (opIndexOrNode as Identifier)
//   }
//   else if (
//     ref is TmplAstTemplate && directive == null &&
//     this.templateCtxOpMap.contains(ref)) {
//     // Resolving the context of the given sub-template.
//     // Execute the `TcbTemplateContextOp` for the template.
//     return this.resolveOp(this.templateCtxOpMap[ref]!!)
//   }
//   else if (
//     (ref is TmplAstElement || ref is TmplAstTemplate) &&
//     directive != null && this.directiveOpMap.contains(ref)) {
//     // Resolving a directive on an element or sub-template.
//     val dirMap = this.directiveOpMap[ref]!!
//     if (dirMap.contains(directive)) {
//       return this.resolveOp(dirMap[directive]!!)
//     }
//     else {
//       return null
//     }
//   }
//   else if (ref is TmplAstElement && this.elementOpMap.contains(ref)) {
//     // Resolving the DOM node of an element in this template.
//     return this.resolveOp(this.elementOpMap[ref]!!)
//   }
//   else {
//     return null
//   }
// }
//
// /**
//  * Like `executeOp`, but assert that the operation actually returned `Expression`.
//  */
// private fun resolveOp(opIndex: Int): Identifier {
//   val res = this.executeOp(opIndex, /* skipOptional */ false)
//   if (res == null) {
//     throw Error("Error resolving operation, got null")
//   }
//   return res
// }
//
// /**
//  * Execute a particular `TcbOp` in the `opQueue`.
//  *
//  * This method replaces the operation in the `opQueue` with the result of execution (once done)
//  * and also protects against a circular dependency from the operation to itself by temporarily
//  * setting the operation's result to a special expression.
//  */
// private fun executeOp(opIndex: Int, skipOptional: Boolean): Identifier? {
//   val op = this.opQueue[opIndex]
//   if (op == null) return op
//   if (op is Identifier) return op
//   if (op !is TcbOp) throw IllegalStateException(op.javaClass.toString())
//
//   if (skipOptional && op.optional) {
//     return null
//   }
//
//   // Set the result of the operation in the queue to its circular fallback. If executing this
//   // operation results in a circular dependency, this will prevent an infinite loop and allow for
//   // the resolution of such cycles.
//   this.opQueue[opIndex] = op.circularFallback()
//   val res = op.execute()
//   // Once the operation has finished executing, it's safe to cache the real result.
//   this.opQueue[opIndex] = res
//   return res
// }
//
// private fun appendNode(node: TmplAstNode) {
//   if (node is TmplAstElement) {
//     this.opQueue.add(TcbElementOp(this.tcb, this, node))
//     this.elementOpMap[node] = this.opQueue.lastIndex
//     if (this.tcb.env.config.controlFlowPreventingContentProjection != ControlFlowPreventingContentProjectionKind.Suppress) {
//       this.appendContentProjectionCheckOp(node)
//     }
//     this.appendDirectivesAndInputsOfNode(node)
//     this.appendOutputsOfNode(node)
//     this.appendChildren(node)
//     this.checkAndAppendReferencesOfNode(node)
//   }
//   else if (node is TmplAstTemplate) {
//     // Template children are rendered in a child scope.
//     this.appendDirectivesAndInputsOfNode(node)
//     this.appendOutputsOfNode(node)
//     this.opQueue.add(TcbTemplateContextOp(this.tcb, this))
//     this.templateCtxOpMap[node] = this.opQueue.lastIndex
//     if (this.tcb.env.config.checkTemplateBodies) {
//       this.opQueue.add(TcbTemplateBodyOp(this.tcb, this, node))
//     }
//     // WebStorm - this is done through HTML validator
//     //else if (this.tcb.env.config.alwaysCheckSchemaInTemplateBodies) {
//     //this.appendDeepSchemaChecks(node.children)
//     //}
//     this.checkAndAppendReferencesOfNode(node)
//   }
//   else if (node is TmplAstDeferredBlock) {
//     this.appendDeferredBlock(node)
//   }
//   else if (node is TmplAstIfBlock) {
//     this.opQueue.add(TcbIfOp(this.tcb, this, node))
//   }
//   else if (node is TmplAstSwitchBlock) {
//     this.opQueue.add(TcbSwitchOp(this.tcb, this, node))
//   }
//   else if (node is TmplAstForLoopBlock) {
//     this.opQueue.add(TcbForOfOp(this.tcb, this, node))
//     if (this.tcb.env.config.checkControlFlowBodies) {
//       this.appendChildren(node.empty)
//     }
//   }
//   else if (node is TmplAstBoundText) {
//     this.opQueue.add(TcbExpressionOp(this.tcb, this, node.value, true))
//   }
//   else if (node is TmplAstContent) {
//     this.appendChildren(node)
//   }
//   else if (node is TmplAstLetBlock) {
//     val declaration = node.declaration
//     if (declaration != null) {
//       this.opQueue.add(TcbLetDeclarationOp(this.tcb, this, declaration))
//       if (this.isLocal(declaration)) {
//         this.tcb.oobRecorder.conflictingDeclaration(this.tcb.id, declaration)
//       }
//       else {
//         this.letDeclOpMap[declaration.name] = LetDeclOpMapRecord(opQueue.lastIndex, declaration)
//       }
//     }
//   }
//   else {
//     throw IllegalStateException("Unsupported node: $node")
//   }
// }
//
// private fun checkAndAppendReferencesOfNode(node: `TmplAstElement|TmplAstTemplate`) {
//   for (ref in node.references.values) {
//     val target = this.tcb.boundTarget.getReferenceTarget(ref)
//
//     if (target == null) {
//       // The reference is invalid if it doesn't have a target, so report it as an error.
//       this.tcb.oobRecorder.missingReferenceTarget(this.tcb.id, ref)
//
//       // Any usages of the invalid reference will be resolved to a variable of type any.
//       this.opQueue.add(TcbInvalidReferenceOp(this.tcb, this))
//     }
//     else {
//       this.opQueue.add(TcbReferenceOp(this.tcb, this, ref, node, target))
//     }
//     this.referenceOpMap[ref] = this.opQueue.lastIndex
//   }
// }
//
// private fun appendDirectivesAndInputsOfNode(node: `TmplAstElement|TmplAstTemplate`) {
//   // Collect all the inputs on the element.
//   val claimedInputs = mutableSetOf<String>()
//   val directives = this.tcb.boundTarget.getDirectivesOfNode(node)
//   if (directives.isEmpty()) {
//     // If there are no directives, then all inputs are unclaimed inputs, so queue an operation
//     // to add them if needed.
//     if (node is TmplAstElement) {
//       this.opQueue.add(TcbUnclaimedInputsOp(this.tcb, this, node, claimedInputs))
//
//       // WebStorm - this is done through HTML validator
//       //this.opQueue.add(
//       //  TcbDomSchemaCheckerOp(this.tcb, node, /* checkElement */ true, claimedInputs))
//     }
//     return
//   }
//   else {
//     if (node is TmplAstElement) {
//       val isDeferred = this.tcb.boundTarget.isDeferred(node)
//       if (!isDeferred && directives.any { dir -> this.tcb.env.isExplicitlyDeferred(dir) }) {
//         // This node has directives/components that were defer-loaded (included into
//         // `@Component.deferredImports`), but the node itself was used outside of a
//         // `@defer` block, which is the error.
//         this.tcb.oobRecorder.deferredComponentUsedEagerly(this.tcb.id, node)
//       }
//     }
//   }
//
//   val dirMap = mutableMapOf<TmplDirectiveMetadata, Int>()
//   for (dir in directives) {
//     var directiveOp: TcbOp
//
//     if (!dir.isGeneric) {
//       // The most common case is that when a directive is not generic, we use the normal
//       // `TcbNonDirectiveTypeOp`.
//       directiveOp = TcbNonGenericDirectiveTypeOp(this.tcb, this, node, dir)
//     }
//     else if (
//       !requiresInlineTypeCtor(dir.typeScriptClass, this.tcb.env) ||
//       this.tcb.env.config.useInlineTypeConstructors) {
//       // For generic directives, we use a type constructor to infer types. If a directive requires
//       // an inline type constructor, then inlining must be available to use the
//       // `TcbDirectiveCtorOp`. If not we, we fallback to using `any` – see below.
//       directiveOp = TcbDirectiveCtorOp(this.tcb, this, node, dir)
//     }
//     else {
//       // If inlining is not available, then we give up on inferring the generic params, and use
//       // `any` type for the directive's generic parameters.
//       directiveOp = TcbGenericDirectiveTypeWithAnyParamsOp(this.tcb, this, node, dir)
//     }
//
//     this.opQueue.add(directiveOp)
//     dirMap[dir] = this.opQueue.lastIndex
//
//     this.opQueue.add(TcbDirectiveInputsOp(this.tcb, this, node, dir))
//   }
//   this.directiveOpMap[node] = dirMap
//
//   // After expanding the directives, we might need to queue an operation to check any unclaimed
//   // inputs.
//   if (node is TmplAstElement) {
//     // Go through the directives and remove any inputs that it claims from `elementInputs`.
//     for (dir in directives) {
//       for (propertyName in dir.inputs.keys) {
//         claimedInputs.add(propertyName)
//       }
//     }
//
//     this.opQueue.add(TcbUnclaimedInputsOp(this.tcb, this, node, claimedInputs))
//     // If there are no directives which match this element, then it's a "plain" DOM element (or a
//     // web component), and should be checked against the DOM schema. If any directives match,
//     // we must assume that the element could be custom (either a component, or a directive like
//     // <router-outlet>) and shouldn't validate the element name itself.
//
//     // WebStorm - this is done through HTML validator
//
//     //val checkElement = directives.isEmpty()
//     //this.opQueue.add(TcbDomSchemaCheckerOp(this.tcb, node, checkElement, claimedInputs))
//   }
// }
//
// private fun appendOutputsOfNode(node: `TmplAstElement|TmplAstTemplate`) {
//   // Collect all the outputs on the element.
//   val claimedOutputs = mutableSetOf<String>()
//   val directives = this.tcb.boundTarget.getDirectivesOfNode(node)
//   if (directives.isEmpty()) {
//     // If there are no directives, then all outputs are unclaimed outputs, so queue an operation
//     // to add them if needed.
//     if (node is TmplAstElement) {
//       this.opQueue.add(TcbUnclaimedOutputsOp(this.tcb, this, node, claimedOutputs))
//     }
//     return
//   }
//
//   // Queue operations for all directives to check the relevant outputs for a directive.
//   for (dir in directives) {
//     this.opQueue.add(TcbDirectiveOutputsOp(this.tcb, this, node, dir))
//   }
//
//   // After expanding the directives, we might need to queue an operation to check any unclaimed
//   // outputs.
//   if (node is TmplAstElement) {
//     // Go through the directives and register any outputs that it claims in `claimedOutputs`.
//     for (dir in directives) {
//       for (outputProperty in dir.outputs.keys) {
//         claimedOutputs.add(outputProperty)
//       }
//     }
//
//     this.opQueue.add(TcbUnclaimedOutputsOp(this.tcb, this, node, claimedOutputs))
//   }
// }
//
// private fun appendContentProjectionCheckOp(root: TmplAstElement) {
//   val meta =
//     this.tcb.boundTarget.getDirectivesOfNode(root).firstNotNullOfOrNull { it.directive as? Angular2Component }
//
//   if (meta?.ngContentSelectors != null && meta.ngContentSelectors.isNotEmpty()) {
//     val selectors = meta.ngContentSelectors
//
//     // We don't need to generate anything for components that don't have projection
//     // slots, or they only have one catch-all slot (represented by `*`).
//     if (selectors.size > 1 || (selectors.size == 1 && selectors[0].text.trim() != "*")) {
//       this.opQueue.add(
//         TcbControlFlowContentProjectionOp(this.tcb, root, selectors, meta.entitySourceName))
//     }
//   }
// }
//
// private fun appendChildren(container: TmplAstNodeWithChildren?) {
//   if (container != null) {
//     for (child in container.children) {
//       this.appendNode(child)
//     }
//   }
// }
//
// private fun appendDeferredBlock(block: TmplAstDeferredBlock) {
//   this.appendDeferredTriggers(block, block.triggers)
//   this.appendDeferredTriggers(block, block.prefetchTriggers)
//   if (block.hydrateTriggers.`when` != null) {
//     this.opQueue.add(TcbExpressionOp(this.tcb, this, block.hydrateTriggers.`when`.value))
//   }
//   this.appendChildren(block)
//   this.appendChildren(block.placeholder)
//   this.appendChildren(block.loading)
//   this.appendChildren(block.error)
// }
//
// private fun appendDeferredTriggers(
//   block: TmplAstDeferredBlock, triggers: TmplAstDeferredBlockTriggers,
// ) {
//   if (triggers.`when` != null) {
//     this.opQueue.add(TcbExpressionOp(this.tcb, this, triggers.`when`.value))
//   }
//   // WebStorm - this is validated through WebStorm inspections
//   //
//   //if (triggers.hover != null) {
//   //  this.appendReferenceBasedDeferredTrigger(block, triggers.hover)
//   //}
//   //
//   //if (triggers.interaction != null) {
//   //  this.appendReferenceBasedDeferredTrigger(block, triggers.interaction)
//   //}
//   //
//   //if (triggers.viewport != null) {
//   //  this.appendReferenceBasedDeferredTrigger(block, triggers.viewport)
//   //}
// }
//
// // WebStorm - this is validated through WebStorm inspections
// //private fun appendReferenceBasedDeferredTrigger(
// //  block: TmplAstDeferredBlock,
// //  trigger: `TmplAstHoverDeferredTrigger|TmplAstInteractionDeferredTrigger|TmplAstViewportDeferredTrigger`,
// //) {
// //  if (this.tcb.boundTarget.getDeferredTriggerTarget(block, trigger) == null) {
// //    this.tcb.oobRecorder.inaccessibleDeferredTriggerElement(this.tcb.id, trigger)
// //  }
// //}
//
//
// /** Reports a diagnostic if there are any `@let` declarations that conflict with a node. */
// private fun checkConflictingLet(node: TmplAstExpressionSymbol) {
//   if (letDeclOpMap.containsKey(node.name)) {
//     tcb.oobRecorder.conflictingDeclaration(
//       tcb.id,
//       letDeclOpMap[node.name]!!.node,
//     )
//   }
// }
