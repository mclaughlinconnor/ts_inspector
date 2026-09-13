package parser

import sitter "github.com/tree-sitter/go-tree-sitter"

// Only used for interesting points

type Function struct {
	BodyNode       *sitter.Node // potentially missing
	IsExport       bool
	Name           string
	NameNode       *sitter.Node
	Node           *sitter.Node
	ParametersNode *sitter.Node
}
