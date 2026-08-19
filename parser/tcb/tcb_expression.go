package tcb

import (
	"slices"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type exprState struct {
	ast          *Ast
	content      []byte
	parts        *Statement
	pipesToClose int
}

var angularExprLang = utils.GetLanguage(utils.AngularExpr)

var exprVisitorFuncMap = walk.NewVisitorFuncsMap[*exprState]()
var exprOptimisedMap walk.VisitorFuncMap[*exprState]

func initTcbExpression() {
	exprVisitorFuncMap["unary_expression"] = visitUnary
	exprVisitorFuncMap["binary_expression"] = visitBinary
	exprVisitorFuncMap["conditional_expression"] = visitBinary
	exprVisitorFuncMap["ternary_expression"] = visitConditional
	exprVisitorFuncMap["template_string"] = visitInterpolation
	exprVisitorFuncMap["bracket_expression"] = visitBracketExpression
	exprVisitorFuncMap["group"] = visitGroup
	exprVisitorFuncMap["array"] = visitLiteralArray
	exprVisitorFuncMap["array"] = visitLiteralArray
	exprVisitorFuncMap["object"] = visitLiteralMap
	exprVisitorFuncMap["string"] = visitString
	exprVisitorFuncMap["number"] = visitNumber
	exprVisitorFuncMap["member_expression"] = visitMemberExpression // visitPropertyRead and visitSafePropertyRead
	exprVisitorFuncMap["assignment_expression"] = visitWrite        // visitPropertyWrite and visitKeyedWrite
	exprVisitorFuncMap["call_expression"] = visitCall
	exprVisitorFuncMap["identifier"] = visitIdentifier
	exprVisitorFuncMap["expression"] = visitExpression
	exprVisitorFuncMap["pipe_sequence"] = visitPipeSequence
	exprVisitorFuncMap["non_null_assertion"] = visitNonNullAssertion

	exprOptimisedMap = walk.GenerateSymbolMap(angularExprLang, exprVisitorFuncMap)
}

func buildTcbExpression(ast *Ast, expression string) *Statement {
	content := []byte(expression)

	root, err := utils.ParseText(content, utils.AngularExpr)
	if err != nil {
		panic(err)
	}

	state := exprState{ast: ast, content: content, parts: &Statement{}}
	output := newWalk(root, &state)
	s := output.parts

	return s
}

func newWalk(node *sitter.Node, state *exprState) *exprState {
	newState := exprState{ast: state.ast, content: state.content, parts: &Statement{}, pipesToClose: 0}

	return walk.VisitNode(node, &newState, 0, exprOptimisedMap, false)
}

func visitUnary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	exprNode := node.ChildByFieldName("value")
	expr := newWalk(exprNode, state)

	state.parts.AddRealPart(operator, operatorNode)
	state.parts.AddStatement(expr.parts)

	return state
}

func visitBinary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	lhsNode := node.ChildByFieldName("left")
	lhs := newWalk(lhsNode, state)

	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	rhsNode := node.ChildByFieldName("right")
	rhs := newWalk(rhsNode, state)

	state.parts.AddStatement(lhs.parts)
	state.parts.AddRealPart(operator, operatorNode)
	state.parts.AddStatement(rhs.parts)

	return state
}

func visitBracketExpression(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	operator := node.ChildByFieldName("object").NextSibling()
	switch operator.Type() {
	default:
		fallthrough
	case ".":
		return visitKeyedRead(node, state, indexInParent, internalFuncMap)
	case "?.":
		return visitSafeKeyedRead(node, state, indexInParent, internalFuncMap)
	case "!.":
		return visitKeyedNonNullAssertRead(node, state, indexInParent, internalFuncMap)
	}
}

func visitGroup(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	// "(", stuff, ")"
	stuffNode := node.NamedChild(0)
	stuff := newWalk(stuffNode, state)

	state.parts.AddVirtPart("(")
	state.parts.AddStatement(stuff.parts)
	state.parts.AddVirtPart(")")

	return state
}

// Chains unsupported in parser
// visitChain(ast: Chain): ts.Expression {
//   const elements = ast.expressions.map(expr => this.translate(expr));
//   const node = wrapForDiagnostics(ts.factory.createCommaListExpression(elements));
//   addParseSpanInfo(node, ast.sourceSpan);
//   return node;
// }

func visitConditional(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	condNode := node.ChildByFieldName("condition")
	condState := newWalk(condNode, state)

	trueNode := node.ChildByFieldName("consequence")
	trueState := newWalk(trueNode, state)

	// Wrap `falseExpr` in parens so that the trailing parse span info is not attributed to the
	// whole conditional.
	// In the following example, the last source span comment (5,6) could be seen as the
	// trailing comment for _either_ the whole conditional expression _or_ just the `falseExpr` that
	// is immediately before it:
	// `conditional /*1,2*/ ? trueExpr /*3,4*/ : falseExpr /*5,6*/`
	// This should be instead be `conditional /*1,2*/ ? trueExpr /*3,4*/ : (falseExpr /*5,6*/)`

	falseState := newWalk(node.ChildByFieldName("alternative"), state)

	state.parts.AddStatement(condState.parts)
	state.parts.AddVirtPart(" ? ")
	state.parts.AddStatement(trueState.parts)
	state.parts.AddVirtPart(" : ")
	state.parts.AddStatement(falseState.parts)

	return state
}

// visitImplicitReceiver(ast: ImplicitReceiver): never {
//   throw new Error('Method not implemented.');
// }

// visitThisReceiver(ast: ThisReceiver): never {
//   throw new Error('Method not implemented.');
// }

func visitInterpolation(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	// Build up a chain of binary + operations to simulate the string concatenation of the
	// interpolation's expressions. The chain is started using an actual string literal to ensure
	// the type is inferred as 'string'.

	for i := range node.NamedChildCount() {
		part := node.NamedChild(int(i))

		partState := newWalk(part, state)
		partExpr := partState.parts

		if i == 0 {
			state.parts.AddVirtPart("''")
		}

		state.parts.AddVirtPart(" + ")
		state.parts.AddStatement(partExpr)
	}

	return state
}

func visitKeyedRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart("[")
	state.parts.AddStatement(keyState.parts)
	state.parts.AddVirtPart("]")

	return state
}

func visitLiteralArray(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.parts.AddVirtPart("[")

	for i := range node.NamedChildCount() {
		if i != 0 {
			state.parts.AddVirtPart(", ")
		}

		element := node.NamedChild(int(i))
		elementState := newWalk(element, state)
		elementState.parts.OffsetByNodeStart(node)
		state.parts.AddStatement(elementState.parts)
	}

	state.parts.AddVirtPart("]")

	// If strictLiteralTypes is disabled, array literals are cast to `any`.
	// const node = this.config.strictLiteralTypes ? literal : tsCastToAny(literal);

	return state
}

func visitLiteralMap(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.parts.AddVirtPart("{")

	for i := range node.NamedChildCount() {
		pair := node.NamedChild(int(i))

		keyNode := pair.ChildByFieldName("key")
		if keyNode == nil {
			continue
		}

		key := "\"" + keyNode.Content(state.content) + "\""

		if i != 0 {
			state.parts.AddVirtPart(", ")
		}

		state.parts.AddRealPart(key, keyNode)
		state.parts.AddVirtPart(": ")

		valueNode := pair.ChildByFieldName("value")
		var valueExpr *Statement
		if valueNode != nil {
			expr := newWalk(valueNode, state)
			valueExpr = expr.parts
		} else {
			valueExpr = StatementFromString(NULL_AS_ANY)
		}

		state.parts.AddStatement(valueExpr)
	}

	state.parts.AddVirtPart("}")

	// If strictLiteralTypes is disabled, object literals are cast to `any`.
	// const node = this.config.strictLiteralTypes ? literal : tsCastToAny(literal);

	return state
}

// visitLiteralPrimitive
// Covered by visitIdentifier
func visitUndefined(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.parts.AddRealPart("undefined", node)

	return state
}

// visitLiteralPrimitive
// Covered by visitIdentifier
func visitNull(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.parts.AddRealPart("null", node)

	return state
}

// visitLiteralPrimitive
func visitString(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.parts.AddRealPart(node.Content(state.content), node)

	return state
}

// visitLiteralPrimitive
func visitNumber(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.parts.AddRealPart(node.Content(state.content), node)

	return state
}

// visitLiteralPrimitive
// Covered by visitIdentifier
// func visitBoolean(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
// 	state.sb.WriteString(node.Content(state.content))
//
// 	return state
// }

func visitMemberExpression(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	operator := node.ChildByFieldName("object").NextSibling()
	switch operator.Type() {
	case ".":
		visitPropertyRead(node, state, indexInParent, internalFuncMap)
	case "?.":
		visitSafePropertyRead(node, state, indexInParent, internalFuncMap)
	case "!.":
		visitNonNullAssertRead(node, state, indexInParent, internalFuncMap)
	}

	return state
}

func visitKeyedNonNullAssertRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart("!")
	state.parts.AddVirtPart("[")
	state.parts.AddStatement(keyState.parts)
	state.parts.AddVirtPart("]")

	return state
}

// visitNonNullAssert
func visitNonNullAssertRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	// This is a normal property read - convert the receiverNode to an expression and emit the correct
	// TypeScript expression to read the property.
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)

	nameNode := node.ChildByFieldName("property")
	name := StatementFromNodeContent(nameNode, state.content)

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart("!.")
	state.parts.AddStatement(name)

	return state
}

// visitPipe(ast: BindingPipe): never {
//   throw new Error('Method not implemented.');
// }

// Covered by visitUnary already
// func visitPrefixNot(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
// 	expressionNode := node.ChildByFieldName("value")
// 	expression := wrapForDiagnostics(newWalk(expressionNode, state).sb.String())
//
// 	state.sb.WriteString("!")
// 	state.sb.WriteString(expression)
//
// 	return state
// }

func visitPropertyRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	// This is a normal property read - convert the receiverNode to an expression and emit the correct
	// TypeScript expression to read the property.
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)

	nameNode := node.ChildByFieldName("property")
	name := StatementFromNodeContent(nameNode, state.content)

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart(".")
	state.parts.AddStatement(name)

	return state
}

// visitPropertyWrite and visitKeyedWrite
func visitWrite(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	leftNode := node.ChildByFieldName("name")
	left := newWalk(leftNode, state).parts

	// The right needs to be wrapped in parens as well or we cannot accurately match its
	// span to just the RHS. For example, the span in `e = $event /*0,10*/` is ambiguous.
	// It could refer to either the whole binary expression or just the RHS.
	// We should instead generate `e = ($event /*0,10*/)` so we know the span 0,10 matches RHS.
	rightNode := node.ChildByFieldName("value")
	right := newWalk(rightNode, state).parts

	state.parts.AddVirtPart("(")
	state.parts.AddStatement(left)
	state.parts.AddVirtPart(" = ")
	state.parts.AddStatement(right)
	state.parts.AddVirtPart(")")

	return state
}

func visitSafePropertyRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)
	receiver := receiverState.parts

	nameNode := node.ChildByFieldName("property")
	name := StatementFromNodeContent(nameNode, state.content)

	// The form of safe property reads depends on whether strictness is in use.
	// if (this.config.strictSafeNavigationTypes) {

	// Basically, the return here is either the type of the complete expression with a null-safe
	// property read, or `undefined`. So a ternary is used to create an "or" type:
	// "a?.b" becomes (null as any ? a!.b : undefined)
	// The type of this expression is (typeof a!.b) | undefined, which is exactly as desired.
	expr := &Statement{}
	expr.AddStatement(receiver)
	expr.AddVirtPart("!.")
	expr.AddStatement(name)

	output := &Statement{}
	output.AddVirtPart("(")
	output.AddVirtPart(NULL_AS_ANY)
	output.AddVirtPart(" ? ")
	output.AddStatement(expr)
	output.AddVirtPart(" : ")
	output.AddVirtPart(UNDEFINED)
	output.AddVirtPart(")")

	state.parts.AddStatement(output)

	// } else if (VeSafeLhsInferenceBugDetector.veWillInferAnyFor(ast)) {
	//   // Emulate a View Engine bug where 'any' is inferred for the left-hand side of the safe
	//   // navigation operation. With this bug, the type of the left-hand side is regarded as any.
	//   // Therefore, the left-hand side only needs repeating in the output (to validate it), and then
	//   // 'any' is used for the rest of the expression. This is done using a comma operator:
	//   // "a?.b" becomes (a as any).b, which will of course have type 'any'.
	//   node = ts.factory.createPropertyAccessExpression(tsCastToAny(receiver), ast.name);
	// } else {
	//   // The View Engine bug isn't active, so check the entire type of the expression, but the final
	//   // result is still inferred as `any`.
	//   // "a?.b" becomes (a!.b as any)
	//   const expr = ts.factory.createPropertyAccessExpression(
	//       ts.factory.createNonNullExpression(receiver), ast.name);
	//   addParseSpanInfo(expr, ast.nameSpan);
	//   node = tsCastToAny(expr);
	// }

	return state
}

func visitSafeKeyedRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)
	receiver := receiverState.parts

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)
	key := keyState.parts

	// The form of safe property reads depends on whether strictness is in use.
	// if (this.config.strictSafeNavigationTypes) {

	// "a?.[...]" becomes (null as any ? a![...] : undefined)

	expr := &Statement{}
	expr.AddStatement(receiver)
	expr.AddVirtPart("![")
	expr.AddStatement(key)
	expr.AddVirtPart("]")

	output := &Statement{}
	output.AddVirtPart("(")
	output.AddVirtPart(NULL_AS_ANY)
	output.AddVirtPart(" ? ")
	output.AddStatement(expr)
	output.AddVirtPart(" : ")
	output.AddVirtPart(UNDEFINED)
	output.AddVirtPart(")")

	// } else if (VeSafeLhsInferenceBugDetector.veWillInferAnyFor(ast)) {
	//   // "a?.[...]" becomes (a as any)[...]
	//   node = ts.factory.createElementAccessExpression(tsCastToAny(receiver), key);
	// } else {
	//   // "a?.[...]" becomes (a!.[...] as any)
	//   const expr = ts.factory.createElementAccessExpression(
	//       ts.factory.createNonNullExpression(receiver), key);
	//   addParseSpanInfo(expr, ast.sourceSpan);
	//   node = tsCastToAny(expr);
	// }

	state.parts.AddStatement(output)

	return state
}

func visitCall(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	argsNode := node.ChildByFieldName("arguments")
	args := make([]*Statement, 0)
	if argsNode != nil {
		argCount := argsNode.NamedChildCount()
		args = make([]*Statement, argCount)

		for i := range argsNode.NamedChildCount() {
			expr := argsNode.NamedChild(int(i))
			s := newWalk(expr, state)
			args[i] = s.parts
		}
	}

	// let expr: ts.Expression;
	receiverNode := node.ChildByFieldName("function")
	receiverStatement := newWalk(receiverNode, state)
	receiverStatementExpr := receiverStatement.parts

	if receiverNode.Content(state.content) == "$any" && len(args) == 1 {
		state.parts.AddVirtPart("(")
		state.parts.AddStatement(args[0])
		state.parts.AddVirtPart(" as any)")

		return state
	}

	chainNode := receiverNode.NextSibling()
	chain := chainNode.Type()

	// // For calls that have a property read as receiver, we have to special-case their emit to avoid
	// // inserting superfluous parenthesis as they prevent TypeScript from applying a narrowing effect
	// // if the method acts as a type guard.
	// if (receiver instanceof PropertyRead) {
	//   const resolved = this.maybeResolve(receiver);
	//   if (resolved !== null) {
	//     expr = resolved;
	//   } else {
	//     const propertyReceiver = wrapForDiagnostics(this.translate(receiver.receiver));
	//     expr = ts.factory.createPropertyAccessExpression(propertyReceiver, receiver.name);
	//     addParseSpanInfo(expr, receiver.nameSpan);
	//   }
	// } else {
	//   expr = this.translate(receiver);
	// }
	//
	// let node: ts.Expression;

	// Safe property/keyed reads will produce a ternary whose value is nullable.
	// We have to generate a similar ternary around the call.
	if chain == "?." {
		state.parts.AddStatement(convertToSafeCall(receiverStatementExpr, args))
	} else {
		state.parts.AddStatement(receiverStatementExpr)

		if chain == "!." {
			state.parts.AddRealPart("!.", chainNode)
		}

		state.parts.AddVirtPart("(")
		for i, arg := range args {
			if i != 0 {
				state.parts.AddVirtPart(", ")
			}

			state.parts.AddStatement(arg)
		}
		state.parts.AddVirtPart(")")
	}

	return state
}

func visitIdentifier(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	name := node.Content(state.content)
	if isIntrinsicValue(name) {
		state.parts.AddRealPart(node.Content(state.content), node)
		return state
	}

	variable := state.ast.Tcb.CurrentScope.GetVariableByName(name)
	if variable != nil {
		state.parts.AddRealPart(variable.Identifier, node)
		return state
	}

	tr := state.ast.FindTagByTemplateRef(name)
	if tr != nil {
		tr.Tag.AddDeclaration(true, true)
		state.parts.AddRealPart(tr.Identifier, node)
		return state
	}

	state.parts.AddVirtPart("this.")
	state.parts.AddRealPart(name, node)

	return state
}

func visitExpression(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	pipesNode := node.ChildByFieldName("pipes")
	if pipesNode != nil {
		walk.VisitNode(pipesNode, state, 0, exprOptimisedMap, false)
	}

	exprNode := node.NamedChild(0)
	if exprNode != nil {
		walk.VisitNode(exprNode, state, 0, exprOptimisedMap, false)
	}

	if pipesNode != nil {
		for _ = range state.pipesToClose {
			state.parts.AddVirtPart(")")
		}
	}

	return state
}

func visitPipeSequence(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	pipeNodes := []*sitter.Node{}

	for i := range node.NamedChildCount() {
		childNode := node.NamedChild(int(i))
		if childNode.Type() != "pipe_call" {
			continue
		}

		pipeNodes = append(pipeNodes, childNode)
	}

	slices.Reverse(pipeNodes)

	tcb := state.ast.Tcb

	for _, pipeNode := range pipeNodes {
		statement := renderPipeNode(pipeNode, state.content, tcb)
		if statement.sb.Len() == 0 {
			continue
		}

		state.parts.AddStatement(statement)
		state.pipesToClose++
	}

	return state
}

func visitNonNullAssertion(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.parts.AddVirtPart("!")

	return state
}

// Covered by visitCall in my AST
// visitSafeCall(ast: SafeCall): ts.Expression {
//   const args = ast.args.map(expr => this.translate(expr));
//   const expr = wrapForDiagnostics(this.translate(ast.receiver));
//   const node = this.convertToSafeCall(ast, expr, args);
//   addParseSpanInfo(node, ast.sourceSpan);
//   return node;
// }

func convertToSafeCall(expr *Statement, args []*Statement) *Statement {
	// if (this.config.strictSafeNavigationTypes) {

	// "a?.method(...)" becomes (null as any ? a!.method(...) : undefined)
	nonNullExpr := &Statement{}
	nonNullExpr.AddStatement(expr)
	nonNullExpr.AddVirtPart("!")

	call := &Statement{}
	call.AddStatement(nonNullExpr)
	call.AddVirtPart("(")
	for i, arg := range args {
		if i != 0 {
			call.AddVirtPart(", ")
		}

		call.AddStatement(arg)
	}
	call.AddVirtPart(")")

	condExpr := &Statement{}
	condExpr.AddVirtPart("(")
	condExpr.AddVirtPart(NULL_AS_ANY)
	condExpr.AddVirtPart(" ? ")
	condExpr.AddStatement(call)
	condExpr.AddVirtPart(" : ")
	condExpr.AddVirtPart(UNDEFINED)
	condExpr.AddVirtPart(")")

	output := &Statement{}
	output.AddVirtPart("(")
	output.AddStatement(condExpr)
	output.AddVirtPart(")")

	return output

	// }

	// if (VeSafeLhsInferenceBugDetector.veWillInferAnyFor(ast)) {
	//   // "a?.method(...)" becomes (a as any).method(...)
	//   return ts.factory.createCallExpression(tsCastToAny(expr), undefined, args);
	// }
	//
	// // "a?.method(...)" becomes (a!.method(...) as any)
	// return tsCastToAny(
	//     ts.factory.createCallExpression(ts.factory.createNonNullExpression(expr), undefined, args));
}

func isIntrinsicValue(text string) bool {
	switch text {
	case "true":
		fallthrough
	case "false":
		fallthrough
	case "null":
		fallthrough
	case "undefined":
		fallthrough
	case "NaN":
		return true
	default:
		return false
	}
}
