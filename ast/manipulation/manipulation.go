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
	funcMap["abstract_class_declaration"] = visitClassDeclaration
	funcMap["accessibility_modifier"] = visitAccessibilityModifier
	funcMap["arguments"] = visitArguments
	funcMap["arrow_function"] = visitArrowFunction
	funcMap["await_expression"] = visitAwaitExpression
	funcMap["binary_expression"] = visitBinaryExpression
	funcMap["break_statement"] = visitBreak
	funcMap["call_expression"] = visitCallExpression
	funcMap["class_declaration"] = visitClassDeclaration
	funcMap["class_heritage"] = visitClassHeritage
	funcMap["continue_statement"] = visitContinue
	funcMap["decorator"] = visitDecorator
	funcMap["else_clause"] = visitElseClause
	funcMap["export_statement"] = visitExportStatement
	funcMap["expression_statement"] = visitExpressionStatement
	funcMap["extends_clause"] = visitExtendsClause
	funcMap["false"] = visitBoolean
	funcMap["for_in_statement"] = visitForInStatement
	funcMap["formal_parameters"] = visitFormalParameters
	funcMap["function_declaration"] = visitFunctionDeclaration
	funcMap["function_signature"] = visitFunctionSignature
	funcMap["identifier"] = visitIdentifier
	funcMap["if_statement"] = visitIfStatement
	funcMap["implements_clause"] = visitImplementsClause
	funcMap["interface_declaration"] = visitInterfaceDeclaration
	funcMap["lexical_declaration"] = visitlexicalDeclaration
	funcMap["member_expression"] = visitMemberExpression
	funcMap["method_definition"] = visitMethodDefinition
	funcMap["method_signature"] = visitMethodSignature
	funcMap["optional_parameter"] = visitParameter
	funcMap["parenthesized_expression"] = visitParenthesizedExpression
	funcMap["program"] = visitProgram
	funcMap["property_identifier"] = visitPropertyIdentifier
	funcMap["public_field_definition"] = visitPublicFieldDefinition
	funcMap["required_parameter"] = visitParameter
	funcMap["return_statement"] = visitReturn
	funcMap["subscript_expression"] = visitSubscriptExpression
	funcMap["this"] = visitThis
	funcMap["true"] = visitBoolean
	funcMap["type"] = visitType
	funcMap["type_identifier"] = visitIdentifier
	funcMap["type_parameter"] = visitTypeParameter
	funcMap["type_parameters"] = visitTypeParameters
	funcMap["unary_expression"] = visitUnaryExpression
	funcMap["variable_declaration"] = visitVariableDeclaration
	funcMap["variable_declarator"] = visitVariableDeclarator
	funcMap["while_statement"] = visitWhileStatement
	funcMap[walk.DUMMY_VISITOR_KIND] = visitUnhandled

	astRoot := Ast{cfg: newCfg(), kind: "root", element: elementFromNode(rootNode), programContent: &programContent{text: byteContent, tree: tree}}
	var ast walkState = &astRoot
	astRoot.getProgramContent().root = ast.(*Ast)

	program, err := walk.WalkTypeScript(rootNode, ast, funcMap)
	if err != nil {
		return nil, err
	}

	astRoot.Program = program

	return &astRoot, nil
}
