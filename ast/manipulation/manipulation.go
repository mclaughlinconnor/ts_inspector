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

func BuildAst(content string) (*root, error) {
	utils.ResetNextId(ID_NAMESPACE)
	byteContent := []byte(content)

	rootNode, tree := utils.ParseTextWithTree(byteContent, utils.TypeScript)

	funcMap := walk.NewVisitorFuncsMap[walkState]()
	funcMap["binary_expression"] = visitBinaryExpression
	funcMap["expression_statement"] = visitExpressionStatement
	funcMap["false"] = visitBoolean
	funcMap["identifier"] = visitIdentifier
	funcMap["program"] = visitProgram
	funcMap["true"] = visitBoolean
	funcMap[walk.DUMMY_VISITOR_KIND] = visitUnhandled

	astRoot := root{commonNode: commonNode{kind: "root", node: rootNode, programContent: &programContent{text: byteContent, tree: tree}}}
	var ast walkState = &astRoot
	astRoot.getProgramContent().root = ast.(*root)

	program, err := walk.WalkTypeScript(rootNode, ast, funcMap)
	if err != nil {
		return nil, err
	}

	astRoot.Program = program

	return &astRoot, nil
}
