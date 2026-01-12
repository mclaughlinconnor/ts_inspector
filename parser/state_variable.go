package parser

import sitter "github.com/smacker/go-tree-sitter"

type Variable struct {
	Kind     string // const/let/var
	IsExport bool
	Name     string
	Node     *sitter.Node
	Value    string
}
