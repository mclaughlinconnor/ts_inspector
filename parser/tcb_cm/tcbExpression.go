package tcb_cm

import (
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type exprState struct {
	content []byte
	parts   *StatementParts
	tcb     *Tcb
}

const NULL_AS_ANY string = "0 as any"
const UNDEFINED string = "undefined"

var angularExprLang = utils.GetLanguage(utils.AngularExpr)

var exprVisitorFuncMap = walk.NewVisitorFuncsMap[*exprState]()
var exprOptimisedMap walk.VisitorFuncMap[*exprState]

func initTcbExpression() {
	exprVisitorFuncMap["unary_expression"] = visitUnary
	exprVisitorFuncMap["binary_expression"] = visitBinary
	exprVisitorFuncMap["ternary_expression"] = visitConditional
	exprVisitorFuncMap["template_string"] = visitInterpolation
	exprVisitorFuncMap["bracket_expression"] = visitBracketExpression
	exprVisitorFuncMap["array"] = visitLiteralArray
	exprVisitorFuncMap["array"] = visitLiteralArray
	exprVisitorFuncMap["string"] = visitString
	exprVisitorFuncMap["number"] = visitNumber
	exprVisitorFuncMap["member_expression"] = visitMemberExpression // visitPropertyRead and visitSafePropertyRead
	exprVisitorFuncMap["assignment_expression"] = visitWrite        // visitPropertyWrite and visitKeyedWrite
	exprVisitorFuncMap["call_expression"] = visitCall
	exprVisitorFuncMap["identifier"] = visitIdentifier

	exprOptimisedMap = walk.GenerateSymbolMap(angularExprLang, exprVisitorFuncMap)
}

func buildTcbExpression(tcb *Tcb, expression string) *StatementParts {
	s, err := utils.ParseText([]byte(expression), utils.AngularExpr, nil, func(root *sitter.Node, content []byte, _ *StatementParts) (*StatementParts, error) {
		state := exprState{content: content, parts: &StatementParts{}, tcb: tcb}
		output := newWalk(root, &state)
		return output.parts, nil
	})

	if err != nil {
		panic(err)
	}

	return s
}

func newWalk(node *sitter.Node, state *exprState) *exprState {
	newState := exprState{content: state.content, parts: &StatementParts{}, tcb: state.tcb}

	return walk.VisitNode(node, &newState, 0, exprOptimisedMap, false)
}

func visitUnary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	exprNode := node.ChildByFieldName("value")
	expr := newWalk(exprNode, state)

	state.parts.AddRealPart(operator, operatorNode)
	state.parts.AddStatementParts(expr.parts)

	return state
}

func visitBinary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	lhsNode := node.ChildByFieldName("left")
	lhs := newWalk(lhsNode, state)

	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	rhsNode := node.ChildByFieldName("right")
	rhs := newWalk(rhsNode, state)

	state.parts.AddStatementParts(lhs.parts)
	state.parts.AddRealPart(operator, operatorNode)
	state.parts.AddStatementParts(rhs.parts)

	return state
}

func visitBracketExpression(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	operator := node.ChildByFieldName("object").NextSibling()
	switch operator.Type() {
	case ".":
		return visitKeyedRead(node, state, indexInParent, internalFuncMap)
	case "?.":
		return visitSafeKeyedRead(node, state, indexInParent, internalFuncMap)
	case "!.":
		return visitKeyedNonNullAssertRead(node, state, indexInParent, internalFuncMap)
	}

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
	wrapForTypeChecker(falseState.parts)

	state.parts.AddStatementParts(condState.parts)
	state.parts.AddVirtPart(" ? ")
	state.parts.AddStatementParts(trueState.parts)
	state.parts.AddVirtPart(" : ")
	state.parts.AddStatementParts(falseState.parts)

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
		state.parts.AddStatementParts(wrapForTypeChecker(partExpr))
	}

	return state
}

func visitKeyedRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)

	state.parts.AddStatementParts(wrapForDiagnostics(receiverState.parts))
	state.parts.AddVirtPart("[")
	state.parts.AddStatementParts(wrapForDiagnostics(keyState.parts))
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
		state.parts.AddStatementParts(wrapForTypeChecker(elementState.parts))
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
		key := "\"" + keyNode.Content(state.content) + "\""

		valueNode := pair.ChildByFieldName("value")
		valueExpr := newWalk(valueNode, state)

		if i != 0 {
			state.parts.AddVirtPart(", ")
		}

		state.parts.AddRealPart(key, keyNode)
		state.parts.AddVirtPart(": ")
		state.parts.AddStatementParts(valueExpr.parts)
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

	state.parts.AddStatementParts(wrapForDiagnostics(receiverState.parts))
	state.parts.AddVirtPart("!")
	state.parts.AddVirtPart("[")
	state.parts.AddStatementParts(wrapForDiagnostics(keyState.parts))
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
	name := wrapForDiagnostics(StatementPartsFromNodeContent(nameNode, state.content))

	state.parts.AddStatementParts(receiverState.parts)
	state.parts.AddVirtPart("!.")
	state.parts.AddStatementParts(name)

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
	name := StatementPartsFromNodeContent(nameNode, state.content)

	state.parts.AddStatementParts(receiverState.parts)
	state.parts.AddVirtPart(".")
	state.parts.AddStatementParts(name)

	return state
}

// visitPropertyWrite and visitKeyedWrite
func visitWrite(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	leftNode := node.ChildByFieldName("name")
	left := wrapForDiagnostics(newWalk(leftNode, state).parts)

	// The right needs to be wrapped in parens as well or we cannot accurately match its
	// span to just the RHS. For example, the span in `e = $event /*0,10*/` is ambiguous.
	// It could refer to either the whole binary expression or just the RHS.
	// We should instead generate `e = ($event /*0,10*/)` so we know the span 0,10 matches RHS.
	rightNode := node.ChildByFieldName("value")
	right := wrapForTypeChecker(newWalk(rightNode, state).parts)

	state.parts.AddVirtPart("(")
	state.parts.AddStatementParts(left)
	state.parts.AddVirtPart(" = ")
	state.parts.AddStatementParts(right)
	state.parts.AddVirtPart(")")

	return state
}

func visitSafePropertyRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)
	receiver := wrapForDiagnostics(receiverState.parts)

	nameNode := node.ChildByFieldName("property")
	name := StatementPartsFromNodeContent(nameNode, state.content)

	// The form of safe property reads depends on whether strictness is in use.
	// if (this.config.strictSafeNavigationTypes) {

	// Basically, the return here is either the type of the complete expression with a null-safe
	// property read, or `undefined`. So a ternary is used to create an "or" type:
	// "a?.b" becomes (null as any ? a!.b : undefined)
	// The type of this expression is (typeof a!.b) | undefined, which is exactly as desired.
	expr := &StatementParts{}
	expr.AddStatementParts(receiver)
	expr.AddVirtPart("!.")
	expr.AddStatementParts(name)

	output := &StatementParts{}
	output.AddVirtPart("(")
	output.AddVirtPart(NULL_AS_ANY)
	output.AddVirtPart(" ? ")
	output.AddStatementParts(expr)
	output.AddVirtPart(" : ")
	output.AddVirtPart(UNDEFINED)
	output.AddVirtPart(")")

	state.parts.AddStatementParts(output)

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
	receiver := wrapForDiagnostics(receiverState.parts)

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)
	key := keyState.parts

	// The form of safe property reads depends on whether strictness is in use.
	// if (this.config.strictSafeNavigationTypes) {

	// "a?.[...]" becomes (null as any ? a![...] : undefined)

	expr := &StatementParts{}
	expr.AddStatementParts(receiver)
	expr.AddVirtPart("![")
	expr.AddStatementParts(key)
	expr.AddVirtPart("]")

	output := &StatementParts{}
	output.AddVirtPart("(")
	output.AddVirtPart(NULL_AS_ANY)
	output.AddVirtPart(" ? ")
	output.AddStatementParts(expr)
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

	state.parts.AddStatementParts(output)

	return state
}

func visitCall(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	argsNode := node.ChildByFieldName("arguments")
	args := make([]*StatementParts, 0)
	if argsNode != nil {
		argCount := argsNode.NamedChildCount()
		args = make([]*StatementParts, argCount)

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
		state.parts.AddStatementParts(args[0])
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
		state.parts.AddStatementParts(convertToSafeCall(receiverStatementExpr, args))
	} else {
		state.parts.AddStatementParts(receiverStatementExpr)

		if chain == "!." {
			state.parts.AddRealPart("!.", chainNode)
		}

		state.parts.AddVirtPart("(")
		for i, arg := range args {
			if i != 0 {
				state.parts.AddVirtPart(", ")
			}

			state.parts.AddStatementParts(arg)
		}
		state.parts.AddVirtPart(")")
	}

	return state
}

func visitIdentifier(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	variable := state.tcb.CurrentScope.GetVariableByName(node.Content(state.content))
	if variable != nil {
		state.parts.AddRealPart(variable.Identifier, node)
	} else {
		state.parts.AddVirtPart("this.")
		state.parts.AddRealPart(node.Content(state.content), node)
	}

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

func convertToSafeCall(expr *StatementParts, args []*StatementParts) *StatementParts {
	// if (this.config.strictSafeNavigationTypes) {

	// "a?.method(...)" becomes (null as any ? a!.method(...) : undefined)
	nonNullExpr := &StatementParts{}
	nonNullExpr.AddStatementParts(expr)
	nonNullExpr.AddVirtPart("!")

	call := &StatementParts{}
	call.AddStatementParts(nonNullExpr)
	call.AddVirtPart("(")
	for i, arg := range args {
		if i != 0 {
			call.AddVirtPart(", ")
		}

		call.AddStatementParts(arg)
	}
	call.AddVirtPart(")")

	condExpr := &StatementParts{}
	condExpr.AddVirtPart("(")
	condExpr.AddVirtPart(NULL_AS_ANY)
	condExpr.AddVirtPart(" ? ")
	condExpr.AddStatementParts(call)
	condExpr.AddVirtPart(" : ")
	condExpr.AddVirtPart(UNDEFINED)
	condExpr.AddVirtPart(")")

	output := &StatementParts{}
	output.AddVirtPart("(")
	output.AddStatementParts(condExpr)
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

func wrapForDiagnostics(expr *StatementParts) *StatementParts {
	expr.PrependVirtPart("(")
	expr.AddVirtPart(")")

	return expr
}

func wrapForTypeChecker(expr *StatementParts) *StatementParts {
	expr.PrependVirtPart("(")
	expr.AddVirtPart(")")

	return expr
}
