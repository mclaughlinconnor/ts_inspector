package manipulation

import (
	"ts_inspector/ast/walk"
	"ts_inspector/utils"
)

const NULL_KIND = "__null_kind"
const ID_NAMESPACE = "manipulation"

const (
	VisitContinue = iota
	VisitAbort    = iota
	VisitSkip     = iota
)

type walkState = nodeInterface

func BuildAst(content string) (*Ast, error) {
	utils.ResetNextId(ID_NAMESPACE)
	byteContent := []byte(content)

	rootNode, tree, err := utils.ParseTextWithTree(byteContent, utils.TypeScript)
	if err != nil {
		return nil, err
	}

	funcMap := walk.NewVisitorFuncsMap[walkState]()
	funcMap["binary_expression"] = visitBinaryExpression
	funcMap["else_clause"] = visitElseClause
	funcMap["expression_statement"] = visitExpressionStatement
	funcMap["false"] = visitBoolean
	funcMap["identifier"] = visitIdentifier
	funcMap["if_statement"] = visitIfStatement
	funcMap["parenthesized_expression"] = visitParenthesizedExpression
	funcMap["program"] = visitProgram
	funcMap["true"] = visitBoolean
	funcMap["unary_expression"] = visitUnaryExpression
	funcMap[walk.DUMMY_VISITOR_KIND] = visitUnhandled

	astRoot := Ast{kind: "root", element: elementFromNode(rootNode), programContent: &programContent{text: byteContent, tree: tree}}
	var ast walkState = &astRoot
	astRoot.getProgramContent().root = ast.(*Ast)

	program, err := walk.WalkTypeScript(rootNode, ast, funcMap)
	if err != nil {
		return nil, err
	}

	astRoot.Program = program

	return &astRoot, nil
}
