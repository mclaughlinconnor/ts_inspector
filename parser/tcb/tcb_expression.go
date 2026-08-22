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
	exprVisitorFuncMap["member_expression"] = visitMemberExpression
	exprVisitorFuncMap["assignment_expression"] = visitWrite
	exprVisitorFuncMap["call_expression"] = visitCall
	exprVisitorFuncMap["identifier"] = visitIdentifier
	exprVisitorFuncMap["expression"] = visitExpression
	exprVisitorFuncMap["pipe_sequence"] = visitPipeSequence
	exprVisitorFuncMap["non_null_assertion"] = visitNonNullAssertion

	exprOptimisedMap = walk.GenerateSymbolMap(angularExprLang, exprVisitorFuncMap)
}

func buildTcbExpression(ast *Ast, expression string) (*Statement, error) {
	content := []byte(expression)

	root, err := utils.ParseText(content, utils.AngularExpr)
	if err != nil {
		return nil, err
	}

	state := exprState{ast: ast, content: content, parts: &Statement{}}
	output, err := newWalk(root, &state)
	if err != nil {
		return nil, err
	}

	return output.parts, nil
}

func newWalk(node *sitter.Node, state *exprState) (*exprState, error) {
	newState := exprState{ast: state.ast, content: state.content, parts: &Statement{}, pipesToClose: 0}

	return walk.VisitNode(node, &newState, 0, exprOptimisedMap, false)
}

func visitUnary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	exprNode := node.ChildByFieldName("value")
	expr, err := newWalk(exprNode, state)
	if err != nil {
		return state, err
	}

	state.parts.AddRealPart(operator, operatorNode)
	state.parts.AddStatement(expr.parts)

	return state, nil
}

func visitBinary(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	lhsNode := node.ChildByFieldName("left")
	lhs, err := newWalk(lhsNode, state)
	if err != nil {
		return state, err
	}

	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(state.content)

	rhsNode := node.ChildByFieldName("right")
	rhs, err := newWalk(rhsNode, state)
	if err != nil {
		return state, err
	}

	state.parts.AddStatement(lhs.parts)
	state.parts.AddRealPart(operator, operatorNode)
	state.parts.AddStatement(rhs.parts)

	return state, nil
}

func visitBracketExpression(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
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

func visitGroup(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	// "(", stuff, ")"
	stuffNode := node.NamedChild(0)
	stuff, err := newWalk(stuffNode, state)
	if err != nil {
		return state, err
	}

	state.parts.AddVirtPart("(")
	state.parts.AddStatement(stuff.parts)
	state.parts.AddVirtPart(")")

	return state, nil
}

func visitConditional(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	condNode := node.ChildByFieldName("condition")
	condState, err := newWalk(condNode, state)
	if err != nil {
		return state, err
	}

	trueNode := node.ChildByFieldName("consequence")
	trueState, err := newWalk(trueNode, state)
	if err != nil {
		return state, err
	}

	falseState, err := newWalk(node.ChildByFieldName("alternative"), state)
	if err != nil {
		return state, err
	}

	state.parts.AddStatement(condState.parts)
	state.parts.AddVirtPart(" ? ")
	state.parts.AddStatement(trueState.parts)
	state.parts.AddVirtPart(" : ")
	state.parts.AddStatement(falseState.parts)

	return state, nil
}

func visitInterpolation(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	for i := range node.NamedChildCount() {
		part := node.NamedChild(int(i))

		partState, err := newWalk(part, state)
		if err != nil {
			return state, err
		}
		partExpr := partState.parts

		if i == 0 {
			state.parts.AddVirtPart("''")
		}

		state.parts.AddVirtPart(" + ")
		state.parts.AddStatement(partExpr)
	}

	return state, nil
}

func visitKeyedRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	receiverNode := node.ChildByFieldName("object")
	receiverState, err := newWalk(receiverNode, state)
	if err != nil {
		return state, err
	}

	keyNode := node.ChildByFieldName("property")
	keyState, err := newWalk(keyNode, state)
	if err != nil {
		return state, err
	}

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart("[")
	state.parts.AddStatement(keyState.parts)
	state.parts.AddVirtPart("]")

	return state, nil
}

func visitLiteralArray(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	state.parts.AddVirtPart("[")

	for i := range node.NamedChildCount() {
		if i != 0 {
			state.parts.AddVirtPart(", ")
		}

		element := node.NamedChild(int(i))
		elementState, err := newWalk(element, state)
		if err != nil {
			return state, err
		}
		elementState.parts.OffsetByNodeStart(node)
		state.parts.AddStatement(elementState.parts)
	}

	state.parts.AddVirtPart("]")

	return state, nil
}

func visitLiteralMap(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
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
			expr, err := newWalk(valueNode, state)
			if err != nil {
				return state, err
			}
			valueExpr = expr.parts
		} else {
			valueExpr = StatementFromString(NULL_AS_ANY)
		}

		state.parts.AddStatement(valueExpr)
	}

	state.parts.AddVirtPart("}")

	return state, nil
}

func visitString(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	state.parts.AddRealPart(node.Content(state.content), node)

	return state, nil
}

func visitNumber(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	state.parts.AddRealPart(node.Content(state.content), node)

	return state, nil
}

func visitMemberExpression(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	var err error

	operator := node.ChildByFieldName("object").NextSibling()
	switch operator.Type() {
	case ".":
		state, err = visitPropertyRead(node, state, indexInParent, internalFuncMap)
	case "?.":
		state, err = visitSafePropertyRead(node, state, indexInParent, internalFuncMap)
	case "!.":
		state, err = visitNonNullAssertRead(node, state, indexInParent, internalFuncMap)
	}

	return state, err
}

func visitKeyedNonNullAssertRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	receiverNode := node.ChildByFieldName("object")
	receiverState, err := newWalk(receiverNode, state)
	if err != nil {
		return state, err
	}

	keyNode := node.ChildByFieldName("property")
	keyState, err := newWalk(keyNode, state)
	if err != nil {
		return state, err
	}

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart("!")
	state.parts.AddVirtPart("[")
	state.parts.AddStatement(keyState.parts)
	state.parts.AddVirtPart("]")

	return state, nil
}

func visitNonNullAssertRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	receiverNode := node.ChildByFieldName("object")
	receiverState, err := newWalk(receiverNode, state)
	if err != nil {
		return state, err
	}

	nameNode := node.ChildByFieldName("property")
	name := StatementFromNodeContent(nameNode, state.content)

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart("!.")
	state.parts.AddStatement(name)

	return state, nil
}

func visitPropertyRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	receiverNode := node.ChildByFieldName("object")
	receiverState, err := newWalk(receiverNode, state)
	if err != nil {
		return state, err
	}

	nameNode := node.ChildByFieldName("property")
	name := StatementFromNodeContent(nameNode, state.content)

	state.parts.AddStatement(receiverState.parts)
	state.parts.AddVirtPart(".")
	state.parts.AddStatement(name)

	return state, nil
}

func visitWrite(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	leftNode := node.ChildByFieldName("name")
	leftState, err := newWalk(leftNode, state)
	if err != nil {
		return state, err
	}

	left := leftState.parts

	rightNode := node.ChildByFieldName("value")
	rightState, err := newWalk(rightNode, state)
	if err != nil {
		return state, err
	}

	right := rightState.parts

	state.parts.AddVirtPart("(")
	state.parts.AddStatement(left)
	state.parts.AddVirtPart(" = ")
	state.parts.AddStatement(right)
	state.parts.AddVirtPart(")")

	return state, nil
}

func visitSafePropertyRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	receiverNode := node.ChildByFieldName("object")
	receiverState, err := newWalk(receiverNode, state)
	if err != nil {
		return state, err
	}

	receiver := receiverState.parts

	nameNode := node.ChildByFieldName("property")
	name := StatementFromNodeContent(nameNode, state.content)

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

	return state, nil
}

func visitSafeKeyedRead(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	receiverNode := node.ChildByFieldName("object")
	receiverState, err := newWalk(receiverNode, state)
	if err != nil {
		return state, err
	}
	receiver := receiverState.parts

	keyNode := node.ChildByFieldName("property")
	keyState, err := newWalk(keyNode, state)
	if err != nil {
		return state, err
	}
	key := keyState.parts

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

	state.parts.AddStatement(output)

	return state, nil
}

func visitCall(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	argsNode := node.ChildByFieldName("arguments")
	args := make([]*Statement, 0)
	if argsNode != nil {
		argCount := argsNode.NamedChildCount()
		args = make([]*Statement, argCount)

		for i := range argsNode.NamedChildCount() {
			expr := argsNode.NamedChild(int(i))
			s, err := newWalk(expr, state)
			if err != nil {
				return state, err
			}
			args[i] = s.parts
		}
	}

	receiverNode := node.ChildByFieldName("function")
	receiverStatement, err := newWalk(receiverNode, state)
	if err != nil {
		return state, err
	}
	receiverStatementExpr := receiverStatement.parts

	if receiverNode.Content(state.content) == "$any" && len(args) == 1 {
		state.parts.AddVirtPart("(")
		state.parts.AddStatement(args[0])
		state.parts.AddVirtPart(" as any)")

		return state, nil
	}

	chainNode := receiverNode.NextSibling()
	chain := chainNode.Type()

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

	return state, nil
}

func visitIdentifier(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	name := node.Content(state.content)
	if isIntrinsicValue(name) {
		state.parts.AddRealPart(node.Content(state.content), node)
		return state, nil
	}

	variable := state.ast.Tcb.CurrentScope.GetVariableByName(name)
	if variable != nil {
		state.parts.AddRealPart(variable.Identifier, node)
		return state, nil
	}

	tr := state.ast.FindTagByTemplateRef(name)
	if tr != nil {
		tr.Tag.AddDeclaration(true, true)
		state.parts.AddRealPart(tr.Identifier, node)
		return state, nil
	}

	state.parts.AddVirtPart("this.")
	state.parts.AddRealPart(name, node)

	return state, nil
}

func visitExpression(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	pipesNode := node.ChildByFieldName("pipes")
	if pipesNode != nil {
		_, err := walk.VisitNode(pipesNode, state, 0, exprOptimisedMap, false)
		if err != nil {
			return nil, err
		}
	}

	exprNode := node.NamedChild(0)
	if exprNode != nil {
		_, err := walk.VisitNode(exprNode, state, 0, exprOptimisedMap, false)
		if err != nil {
			return nil, err
		}
	}

	if pipesNode != nil {
		for range state.pipesToClose {
			state.parts.AddVirtPart(")")
		}
	}

	return state, nil
}

func visitPipeSequence(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
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

	return state, nil
}

func visitNonNullAssertion(node *sitter.Node, state *exprState, indexInParent int, internalFuncMap walk.VisitorFuncMap[*exprState]) (*exprState, error) {
	state.parts.AddVirtPart("!")

	return state, nil
}

func convertToSafeCall(expr *Statement, args []*Statement) *Statement {
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
