package tcb_cm

import (
	"ts_inspector/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

type Attribute struct {
	renderable

	Name        string
	Node        *sitter.Node
	SourceClass *parser.Class
	Value       string
}

func (a *Attribute) Render()
