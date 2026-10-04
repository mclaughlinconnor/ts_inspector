package manipulation

import (
	"ts_inspector/ast/walk"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type commonMemberSubscript struct {
	commonNode
	object          nodeInterface
	isOptionalChain bool
}

func visitCommonMemberSubscript(node *sitter.Node, state walkState, _ uint, funcMap walk.VisitorFuncMap[walkState]) (*commonMemberSubscript, error) {
	commonMemberSubscript := commonMemberSubscript{commonNode: makeCommonNode("commonMemberSubscript", state, node)}
	commonMemberSubscript.setImpl(&commonMemberSubscript)

	objectNode := node.ChildByFieldName("object")
	if objectNode == nil {
		return nil, newAstErrorTS(node, state, "missing object")
	}

	isOptionalChainNode := node.ChildByFieldName("operator")

	object, err := walk.VisitNode(objectNode, state, 0, funcMap, false)
	if err != nil {
		return nil, err
	}

	commonMemberSubscript.object = object
	commonMemberSubscript.isOptionalChain = isOptionalChainNode != nil

	return &commonMemberSubscript, nil
}
