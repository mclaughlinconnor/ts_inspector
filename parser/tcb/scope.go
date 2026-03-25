package tcb

import (
	"strings"
)

type LetDeclOpMapRecord struct {
	opIndex int
	node    *TmplAstNode
}

type IntOrIdentifier struct {
	integer    *int
	identifier *Identifier
}

type TcbOpOrIdentifier struct {
	op         *TcbOp
	identifier *Identifier
}

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
	opQueue []*TcbOpOrIdentifier

	/**
	* A map of `TmplAstElement`s to the index of their `TcbElementOp` in the `opQueue`
	 */
	elementOpMap map[*TmplAstNode]int

	/**
	* A map of maps which tracks the index of `TcbDirectiveCtorOp`s in the `opQueue` for each
	* directive on a `TmplAstElement` or `TmplAstTemplate` node.
	 */
	directiveOpMap map[*TmplAstNode]map[*TmplDirectiveMetadata]int

	/**
	* A map of `TmplAstReference`s to the index of their `TcbReferenceOp` in the `opQueue`
	 */
	referenceOpMap map[*TmplAstNode]int

	/**
	* Map of immediately nested <ng-template>s (within this `Scope`) represented by `TmplAstTemplate`
	* nodes to the index of their `TcbTemplateContextOp`s in the `opQueue`.
	 */
	templateCtxOpMap map[*TmplAstNode]int

	/**
	* Map of variables declared on the template that created this `Scope` (represented by
	* `TmplAstVariable` nodes) to the index of their `TcbVariableOp`s in the `opQueue`, or to
	* pre-resolved variable identifiers.
	 */
	varMap map[*TmplAstNode]IntOrIdentifier

	/**
	* A map of the names of `TmplAstLetDeclaration`s to the index of their op in the `opQueue`.
	*
	* Assumes that there won't be duplicated `@let` declarations within the same scope.
	 */
	letDeclOpMap map[string]LetDeclOpMapRecord

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
func scopeForNodes(tcb *Context, parentScope *Scope, scopedNode *TmplAstNode, children []*TmplAstNode, guard *Expression) Scope {
	var guardExpr *Expression = nil

	if guard != nil {
		guardExpr := Expression{}
		guardExpr = append(guardExpr, "(")
		guardExpr = append(guardExpr, strings.Join(*guard, ""))
		guardExpr = append(guardExpr, ")")
	}

	scope := Scope{
		tcb:              tcb,
		parent:           parentScope,
		guard:            guardExpr,
		elementOpMap:     make(map[*TmplAstNode]int),
		directiveOpMap:   make(map[*TmplAstNode]map[*TmplDirectiveMetadata]int),
		referenceOpMap:   make(map[*TmplAstNode]int),
		templateCtxOpMap: make(map[*TmplAstNode]int),
		varMap:           make(map[*TmplAstNode]IntOrIdentifier),
		letDeclOpMap:     make(map[string]LetDeclOpMapRecord),
	}

	// If given an actual `TmplAstTemplate` instance, then process any additional information it has.
	if scopedNode.Kind == KindTmplAstTemplate {
		// The template"s variable declarations need to be added as `TcbVariableOp`s.
		varMap := map[string]*TmplAstVariable{}

		for _, v := range scopedNode.Variables {
			// Validate that variables on the `TmplAstTemplate` are only declared once.
			variable, found := varMap[v.Name]
			if !found {
				varMap[v.Name] = v
			} else {
				firstDecl := variable
				tcb.oobRecorder.duplicateTemplateVar(tcb.id, v, firstDecl)
			}

			scope.registerVariable(v, TcbTemplateVariableOp{tcb: tcb, scope: &scope, template: scopedNode, variable: v})
		}
	} else if scopedNode.Kind == KindTmplAstIfBlockBranch {
		expression := scopedNode.Expression
		expressionAlias := scopedNode.ExpressionAlias
		if expression != nil && expressionAlias != nil {
			scope.registerVariable(
				expressionAlias,
				TcbBlockVariableOp{
					tcb:         tcb,
					scope:       &scope,
					initializer: Expression{AstToTypescript(expression)},
					variable:    *expressionAlias,
				},
			)
		}
	} else if scopedNode.Kind == KindTmplAstForLoopBlock {
		// Register the variable for the loop so it can be resolved by
		// children. It'll be declared once the loop is created.
		variable := scopedNode.Variable
		if variable != nil {
			loopInitializer := tcb.allocateId(variable, nil)
			scope.varMap[variable] = loopInitializer
		}

		for _, v := range scopedNode.ContextVariables {
			typeName, found := forLoopContextVariableTypes[v.Name]
			if !found {
				typeName = "any"
			}

			ttype := Expression{typeName}
			scope.registerVariable(
				variable,
				TcbBlockImplicitVariableOp{tcb: tcb, scope: &scope, ttype: ttype, variable: variable, initializer: nil},
			)
		}
	}

	for _, node := range children {
		scope.appendNode(node)
	}

	// Once everything is registered, we need to check if there are `@let`
	// declarations that conflict with other local symbols defined after them.
	for variable := range scope.varMap {
		scope.checkConflictingLet(variable.TmplAstExpressionSymbol)
	}

	for ref := range scope.referenceOpMap {
		scope.checkConflictingLet(ref.TmplAstExpressionSymbol)
	}

	return scope
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
/** Registers a local variable with a scope. */
func (s *Scope) registerVariable(variable *TmplAstVariable, op TcbOp) {
	s.opQueue = append(s.opQueue, &op)
	opIndex := len(s.opQueue) - 1
	s.varMap[variable] = opIndex
}

//
/**
 * Look up a `Expression` representing the value of some operation in the current `Scope`,
 * including any parent scope(s). This method always returns a mutable clone of the
 * `Expression` with the comments cleared.
 *
 * @param node a `TmplAstNode` of the operation in question. The lookup performed will depend on
 * the type of this node:
 *
 * Assuming `directive` is not present, then `resolve` will return:
 *
 * * `TmplAstElement` - retrieve the expression for the element DOM node
 * * `TmplAstTemplate` - retrieve the template context variable
 * * `TmplAstVariable` - retrieve a template let- variable
 * * `TmplAstReference` - retrieve variable created for the local ref
 *
 * @param directive if present, a directive type on a `TmplAstElement` or `TmplAstTemplate` to
 * look up instead of the default for an element or template node.
 */
func (s *Scope) resolve(node TmplAstNode /* LocalSymbol */, directive *TmplDirectiveMetadata) *Identifier {
	// Attempt to resolve the operation locally.
	res := s.resolveLocal(node, directive)
	if res != nil {
		return res
	} else if s.parent != nil {
		// Check with the parent.
		return s.parent.resolve(node, directive)
	} else {
		panic("Could not resolve ${node} / ${directive}")
	}
}

/**
 * Add a statement to this scope.
 */
func (s *Scope) addStatementStatement(stmt Statement) {
	s.statements = append(s.statements, stmt)
}

/**
 * Add a statement to this scope.
 */
func (s *Scope) addStatementExpression(expr Expression) {
	s.statements = append(s.statements, Statement{append(expr, ";")})
}

//	fun addStatement(builder: Expression.ExpressionBuilder.() -> Unit) {
//	  addStatement(Statement(builder))
//	}
//
// /**
//
//   - Get the statements.
//     */
//
//     fun render(): List<Statement> {
//     for (i in opQueue.indices) {
//     // Optional statements cannot be skipped when we are generating the TCB for use
//     // by the TemplateTypeChecker.
//     val skipOptional = !this.tcb.env.config.enableTemplateTypeChecker
//     this.executeOp(i, skipOptional)
//     }
//     return this.statements
//     }
//
// /**
//
//   - Returns an expression of all template guards that apply to this scope, including those of
//
//   - parent scopes. If no guards have been applied, null is returned.
//     */
//
//     fun guards(): Expression? {
//     var parentGuards: Expression? = null
//     if (this.parent != null) {
//     // Start with the guards from the parent scope, if present.
//     parentGuards = this.parent.guards()
//     }
//
//     if (this.guard == null) {
//     // This scope does not have a guard, so return the parent's guards as is.
//     return parentGuards
//     }
//     else if (parentGuards == null) {
//     // There's no guards from the parent scope, so this scope's guard represents all available
//     // guards.
//     return this.guard
//     }
//     else {
//     // Both the parent scope and this scope provide a guard, so create a combination of the two.
//     // It is important that the parent guard is used as left operand, given that it may provide
//     // narrowing that is required for this scope's guard to be valid.
//     return Expression {
//     append(parentGuards)
//     append(" && ")
//     append(this@Scope.guard)
//     }
//     }
//     }
//
// /** Returns whether a template symbol is defined locally within the current scope. */
//
//	fun isLocal(node: TemplateEntity): Boolean {
//	  if (node is TmplAstVariable) {
//	    return this.varMap.containsKey(node)
//	  }
//	  if (node is TmplAstLetDeclaration) {
//	    return this.letDeclOpMap.containsKey(node.name)
//	  }
//	  return this.referenceOpMap.containsKey(node)
//	}
func (s *Scope) resolveLocal(ref TmplAstNode /* LocalSymbol */, directive *TmplDirectiveMetadata) *Identifier {
	if _, ok := s.referenceOpMap[&ref]; ref.Kind == KindTmplAstReference && ok {
		return s.resolveOp(s.referenceOpMap[&ref])
	}

	if _, ok := s.letDeclOpMap[ref.Name]; ref.Kind == KindTmplAstLetDeclaration && ok {
		return s.resolveOp(s.letDeclOpMap[ref.Name].opIndex)
	}

	if _, ok := s.varMap[&ref]; ref.Kind == KindTmplAstVariable && ok {
		// Resolving a context variable for s template.
		// Execute the `TcbVariableOp` associated with the `TmplAstVariable`.
		opIndexOrNode := s.varMap[&ref]
		if opIndexOrNode.integer != nil {
			return s.resolveOp(opIndexOrNode)
		} else {
			return opIndexOrNode.identifier
		}
	}

	if _, ok := s.templateCtxOpMap[&ref]; ref.Kind == KindTmplAstTemplate && directive != nil && ok {
		// Resolving the context of the given sub-template.
		// Execute the `TcbTemplateContextOp` for the template.
		return s.resolveOp(s.templateCtxOpMap[&ref])
	}

	if _, ok := s.directiveOpMap[&ref]; (ref.Kind == KindTmplAstElement || ref.Kind == KindTmplAstTemplate) &&
		directive != nil && ok {
		// Resolving a directive on an element or sub-template.
		dirMap := s.directiveOpMap[&ref]
		if _, ok := dirMap[directive]; ok {
			return s.resolveOp(dirMap[directive])
		} else {
			return nil
		}
	}

	if _, ok := s.elementOpMap[&ref]; ref.Kind == KindTmplAstElement && ok {
		// Resolving the DOM node of an element in s template.
		return s.resolveOp(s.elementOpMap[&ref])
	}

	return nil
}

/**
 * Like `executeOp`, but assert that the operation actually returned `Expression`.
 */
func (s *Scope) resolveOp(opIndex int) Identifier {
	res := s.executeOp(opIndex /* skipOptional */, false)
	if res == nil {
		panic("Error resolving operation, got null")
	}

	return res
}

/**
 * Execute a particular `TcbOp` in the `opQueue`.
 *
 * This method replaces the operation in the `opQueue` with the result of execution (once done)
 * and also protects against a circular dependency from the operation to itself by temporarily
 * setting the operation's result to a special expression.
 */
func (s *Scope) executeOp(opIndex int, skipOptional bool) *Identifier {
	op := s.opQueue[opIndex]

	if op == nil {
		return op.identifier
	}
	if op.identifier != nil {
		return op.identifier
	}
	if op.op == nil {
		panic(op)
	}

	o := (*op.op)

	if skipOptional && o.Optional() {
		return nil
	}

	// Set the result of the operation in the queue to its circular fallback. If executing this
	// operation results in a circular dependency, this will prevent an infinite loop and allow for
	// the resolution of such cycles.
	var fallback TcbOp = o.CircularFallback()
	s.opQueue[opIndex] = &TcbOpOrIdentifier{op: &fallback}
	var res *Identifier = o.Execute()
	// Once the operation has finished executing, it's safe to cache the real result.
	s.opQueue[opIndex] = &TcbOpOrIdentifier{identifier: res}
	return res
}

func (s *Scope) appendNode(node *TmplAstNode) {
	if node.Kind == KindTmplAstElement {
		var op TcbOp = TcbElementOp{tcb: s.tcb, scope: s, element: node}
		s.opQueue = append(s.opQueue, &op)
		s.elementOpMap[node] = len(s.opQueue) - 1
		// if s.tcb.env.config.controlFlowPreventingContentProjection != ControlFlowPreventingContentProjectionKind.Suppress {
		s.appendContentProjectionCheckOp(node)
		// }
		s.appendDirectivesAndInputsOfNode(node)
		s.appendOutputsOfNode(node)
		s.appendChildren(node)
		s.checkAndAppendReferencesOfNode(node)
	} else if node.Kind == KindTmplAstTemplate {
		// Template children are rendered in a child scope.
		s.appendDirectivesAndInputsOfNode(node)
		s.appendOutputsOfNode(node)
		contextOp := TcbOp(TcbTemplateContextOp{tcb: s.tcb, scope: s})
		s.opQueue = append(s.opQueue, &contextOp)
		s.templateCtxOpMap[node] = len(s.opQueue) - 1
		// if s.tcb.env.config.checkTemplateBodies {
		bodyOp := TcbOp(TcbTemplateBodyOp{tcb: *s.tcb, scope: *s, template: *node})
		s.opQueue = append(s.opQueue, &bodyOp)
		// }

		// WebStorm - this is done through HTML validator. CM - checkTemplateBodies does it anyway
		//else if (this.tcb.env.config.alwaysCheckSchemaInTemplateBodies) {
		//this.appendDeepSchemaChecks(node.children)
		//}

		s.checkAndAppendReferencesOfNode(node)
	} else if node.Kind == KindTmplAstDeferredBlock {
		s.appendDeferredBlock(node)
	} else if node.Kind == KindTmplAstIfBlock {
		s.opQueue = append(s.opQueue, TcbIfOp{s.tcb, s, node})
	} else if node.Kind == KindTmplAstSwitchBlock {
		s.opQueue = append(s.opQueue, TcbSwitchOp{s.tcb, s, node})
	} else if node.Kind == KindTmplAstForLoopBlock {
		s.opQueue = append(s.opQueue, TcbForOfOp{s.tcb, s, node})
		// if (s.tcb.env.config.checkControlFlowBodies) {
		s.appendChildren(node.empty)
		// }
	} else if node.Kind == KindTmplAstBoundText {
		s.opQueue = append(s.opQueue, TcbExpressionOp{s.tcb, s, node.value, true})
	} else if node.Kind == KindTmplAstContent {
		s.appendChildren(node)
	} else if node.Kind == KindTmplAstLetBlock {
		declaration := node.Declaration
		if declaration != nil {
			s.opQueue.add(TcbLetDeclarationOp{s.tcb, s, declaration})
			if s.isLocal(declaration) {
				s.tcb.oobRecorder.conflictingDeclaration(s.tcb.id, declaration)
			} else {
				s.letDeclOpMap[declaration.name] = LetDeclOpMapRecord(opQueue.lastIndex, declaration)
			}
		}
	} else {
		// throw IllegalStateException("Unsupported node: $node")
	}
}

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

func (s *Scope) appendChildren(container *TmplAstNode /* KindTmplAstNodeWithChildren */) {
	if container == nil {
		return
	}

	for _, child := range container.children {
		s.appendNode(child)
	}
}

func (s *Scope) appendDeferredBlock(block TmplAstNode /* KindTmplAstDeferredBlock */) {
	s.appendDeferredTriggers(block, block.triggers)
	s.appendDeferredTriggers(block, block.prefetchTriggers)

	if block.hydrateTriggers.when != nil {
		var expression TcbOp = TcbExpressionOp{tcb: s.tcb, scope: s, expression: block.hydrateTriggers.when.value, isBoundText: false}
		s.opQueue = append(s.opQueue, &expression)
	}

	s.appendChildren(&block)
	s.appendChildren(block.placeholder)
	s.appendChildren(block.loading)
	s.appendChildren(block.err)
}

func (s *Scope) appendDeferredTriggers(block TmplAstNode /*TmplAstDeferredBlock*/, triggers TmplAstDeferredBlockTriggers) {
	if triggers.when != nil {
		var expression TcbOp = TcbExpressionOp{tcb: s.tcb, scope: s, expression: triggers.when.value, isBoundText: false}
		s.opQueue = append(s.opQueue, &expression)
	}
	// WebStorm - this is validated through WebStorm inspections
	//
	//if (triggers.hover != null) {
	//  this.appendReferenceBasedDeferredTrigger(block, triggers.hover)
	//}
	//
	//if (triggers.interaction != null) {
	//  this.appendReferenceBasedDeferredTrigger(block, triggers.interaction)
	//}
	//
	//if (triggers.viewport != null) {
	//  this.appendReferenceBasedDeferredTrigger(block, triggers.viewport)
	//}
}

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

/** Reports a diagnostic if there are any `@let` declarations that conflict with a node. */
func (s *Scope) checkConflictingLet(node TmplAstExpressionSymbol) {
	op, found := s.letDeclOpMap[node.Name]
	if found {
		s.tcb.oobRecorder.conflictingDeclaration(
			s.tcb.id,
			op.node,
		)
	}
}
