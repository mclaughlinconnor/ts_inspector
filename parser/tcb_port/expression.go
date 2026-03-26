package tcb_port

import (
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type exprState struct {
	sb      *strings.Builder
	content []byte
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

func AstToTypescript(expression *Expression) string {
	content := []byte(strings.Join(*expression, ""))

	s, err := utils.ParseText(content, utils.AngularExpr, "", func(root *sitter.Node, content []byte, _ string) (string, error) {
		state := exprState{content: content, sb: &strings.Builder{}}
		output := newWalk(root, &state)
		return output.sb.String(), nil
	})

	if err != nil {
		panic(err)
	}

	return s
}

func newWalk(node *sitter.Node, state *exprState) *exprState {
	newState := exprState{content: []byte(node.Content(state.content)), sb: &strings.Builder{}}

	return walk.VisitNode(node, &newState, 0, exprOptimisedMap, false)
}

func visitUnary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	exprNode := node.ChildByFieldName("value")
	expr := newWalk(exprNode, state)

	state.sb.WriteString(operator)
	state.sb.WriteString(expr.sb.String())

	return state
}

func visitBinary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	lhsNode := node.ChildByFieldName("left")
	lhs := newWalk(lhsNode, state)

	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	rhsNode := node.ChildByFieldName("right")
	rhs := newWalk(rhsNode, state)

	state.sb.WriteString(lhs.sb.String())
	state.sb.WriteString(operator)
	state.sb.WriteString(rhs.sb.String())

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
	condExpr := condState.sb.String()

	trueNode := node.ChildByFieldName("consequence")
	trueState := newWalk(trueNode, state)
	trueExpr := trueState.sb.String()

	// Wrap `falseExpr` in parens so that the trailing parse span info is not attributed to the
	// whole conditional.
	// In the following example, the last source span comment (5,6) could be seen as the
	// trailing comment for _either_ the whole conditional expression _or_ just the `falseExpr` that
	// is immediately before it:
	// `conditional /*1,2*/ ? trueExpr /*3,4*/ : falseExpr /*5,6*/`
	// This should be instead be `conditional /*1,2*/ ? trueExpr /*3,4*/ : (falseExpr /*5,6*/)`

	falseState := newWalk(node.ChildByFieldName("alternative"), state)
	falseExpr := wrapForTypeChecker(falseState.sb.String())

	state.sb.WriteString(condExpr)
	state.sb.WriteString(" ? ")
	state.sb.WriteString(trueExpr)
	state.sb.WriteString(" : ")
	state.sb.WriteString(falseExpr)

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
		partExpr := partState.sb.String()

		if i == 0 {
			state.sb.WriteString("''")
		}

		state.sb.WriteString(" + ")
		state.sb.WriteString(wrapForTypeChecker(partExpr))
	}

	return state
}

func visitKeyedRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)
	receiver := wrapForDiagnostics(receiverState.sb.String())

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)
	key := wrapForDiagnostics(keyState.sb.String())

	state.sb.WriteString(receiver)
	state.sb.WriteString("[")
	state.sb.WriteString(key)
	state.sb.WriteString("]")

	return state
}

func visitLiteralArray(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.sb.WriteString("[")

	for i := range node.NamedChildCount() {
		element := node.NamedChild(int(i))

		elementState := newWalk(element, state)
		elementExpr := elementState.sb.String()

		if i != 0 {
			state.sb.WriteString(", ")
		}

		state.sb.WriteString(wrapForTypeChecker(elementExpr))
	}

	state.sb.WriteString("]")

	// If strictLiteralTypes is disabled, array literals are cast to `any`.
	// const node = this.config.strictLiteralTypes ? literal : tsCastToAny(literal);

	return state
}

func visitLiteralMap(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.sb.WriteString("{")

	for i := range node.NamedChildCount() {
		pair := node.NamedChild(int(i))

		keyNode := pair.ChildByFieldName("key")
		key := "\"" + keyNode.Content(state.content) + "\""

		valueNode := pair.ChildByFieldName("value")
		valueExpr := newWalk(valueNode, state)
		value := valueExpr.sb.String()

		if i != 0 {
			state.sb.WriteString(", ")
		}

		state.sb.WriteString(key)
		state.sb.WriteString(": ")
		state.sb.WriteString(value)
	}

	state.sb.WriteString("}")

	// If strictLiteralTypes is disabled, object literals are cast to `any`.
	// const node = this.config.strictLiteralTypes ? literal : tsCastToAny(literal);

	return state
}

// visitLiteralPrimitive
// Covered by visitIdentifier
func visitUndefined(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.sb.WriteString("undefined")

	return state
}

// visitLiteralPrimitive
// Covered by visitIdentifier
func visitNull(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.sb.WriteString("null")

	return state
}

// visitLiteralPrimitive
func visitString(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.sb.WriteString(node.Content(state.content))

	return state
}

// visitLiteralPrimitive
func visitNumber(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.sb.WriteString(node.Content(state.content))

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
		return visitPropertyRead(node, state, indexInParent, internalFuncMap)
	case "?.":
		return visitSafePropertyRead(node, state, indexInParent, internalFuncMap)
	case "!.":
		return visitNonNullAssertRead(node, state, indexInParent, internalFuncMap)
	}

	return state
}

func visitKeyedNonNullAssertRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)
	receiver := wrapForDiagnostics(receiverState.sb.String())

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)
	key := wrapForDiagnostics(keyState.sb.String())

	state.sb.WriteString(receiver)
	state.sb.WriteString("!")
	state.sb.WriteString("[")
	state.sb.WriteString(key)
	state.sb.WriteString("]")

	return state
}

// visitNonNullAssert
func visitNonNullAssertRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	// This is a normal property read - convert the receiverNode to an expression and emit the correct
	// TypeScript expression to read the property.
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)
	receiver := receiverState.sb.String()

	nameNode := node.ChildByFieldName("property")
	name := wrapForDiagnostics(nameNode.Content(state.content))

	state.sb.WriteString(receiver)
	state.sb.WriteString("!.")
	state.sb.WriteString(name)

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
	receiver := receiverState.sb.String()

	nameNode := node.ChildByFieldName("property")
	name := wrapForDiagnostics(nameNode.Content(state.content))

	state.sb.WriteString(receiver)
	state.sb.WriteString(".")
	state.sb.WriteString(name)

	return state
}

// visitPropertyWrite and visitKeyedWrite
func visitWrite(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	leftNode := node.ChildByFieldName("name")
	left := wrapForDiagnostics(newWalk(leftNode, state).sb.String())

	// The right needs to be wrapped in parens as well or we cannot accurately match its
	// span to just the RHS. For example, the span in `e = $event /*0,10*/` is ambiguous.
	// It could refer to either the whole binary expression or just the RHS.
	// We should instead generate `e = ($event /*0,10*/)` so we know the span 0,10 matches RHS.
	rightNode := node.ChildByFieldName("value")
	right := wrapForTypeChecker(newWalk(rightNode, state).sb.String())

	state.sb.WriteString("(")
	state.sb.WriteString(left)
	state.sb.WriteString(" = ")
	state.sb.WriteString(right)
	state.sb.WriteString(")")

	return state
}

func visitSafePropertyRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	receiverNode := node.ChildByFieldName("object")
	receiverState := newWalk(receiverNode, state)
	receiver := wrapForDiagnostics(receiverState.sb.String())

	nameNode := node.ChildByFieldName("property")
	name := wrapForDiagnostics(nameNode.Content(state.content))

	// The form of safe property reads depends on whether strictness is in use.
	// if (this.config.strictSafeNavigationTypes) {

	// Basically, the return here is either the type of the complete expression with a null-safe
	// property read, or `undefined`. So a ternary is used to create an "or" type:
	// "a?.b" becomes (null as any ? a!.b : undefined)
	// The type of this expression is (typeof a!.b) | undefined, which is exactly as desired.
	expr := receiver + "!." + name
	output := "(" + NULL_AS_ANY + " ? " + expr + " : " + UNDEFINED + ")"

	state.sb.WriteString(output)

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
	receiver := wrapForDiagnostics(receiverState.sb.String())

	keyNode := node.ChildByFieldName("property")
	keyState := newWalk(keyNode, state)
	key := wrapForDiagnostics(keyState.sb.String())

	// The form of safe property reads depends on whether strictness is in use.
	// if (this.config.strictSafeNavigationTypes) {

	// "a?.[...]" becomes (null as any ? a![...] : undefined)

	expr := receiver + "![" + key + "]"
	output := "(" + NULL_AS_ANY + " ? " + expr + " : " + UNDEFINED + ")"

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

	state.sb.WriteString(output)

	return state
}

func visitCall(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	argsNode := node.ChildByFieldName("arguments")

	argCount := argsNode.NamedChildCount()
	args := make([]string, argCount)

	for i := range argsNode.NamedChildCount() {
		expr := argsNode.NamedChild(int(i))
		s := newWalk(expr, state)
		args[i] = s.sb.String()
	}

	// let expr: ts.Expression;
	receiverNode := node.ChildByFieldName("function")
	receiver := receiverNode.Content(state.content)

	chain := receiverNode.NextSibling().Type()

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
		state.sb.WriteString(convertToSafeCall(receiver, args))
	} else {
		state.sb.WriteString(receiver)

		if chain == "!." {
			state.sb.WriteString("!.")
		}

		state.sb.WriteString("(")
		state.sb.WriteString(strings.Join(args, ", "))
		state.sb.WriteString(")")
	}

	return state
}

func visitIdentifier(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) *exprState {
	state.sb.WriteString(node.Content(state.content))

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

func convertToSafeCall(expr string, args []string) string {
	// if (this.config.strictSafeNavigationTypes) {

	// "a?.method(...)" becomes (null as any ? a!.method(...) : undefined)
	nonNullExpr := expr + "!"
	call := nonNullExpr + "(" + strings.Join(args, ", ") + ")"
	condExpr := NULL_AS_ANY + " ? " + call + " : " + UNDEFINED

	return "(" + condExpr + ")"

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

func wrapForDiagnostics(expr string) string {
	return "(" + expr + ")"
}

func wrapForTypeChecker(expr string) string {
	return "(" + expr + ")"
}
