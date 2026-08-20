package tcb

import (
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func renderPipeNode(node *sitter.Node, content []byte, tcb *Tcb) *Statement {
	class := tcb.Class
	state := tcb.State

	statement := Statement{}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return &statement
	}

	name := nameNode.Content(content)

	for _, thing := range class.Snapshot().Angular.Component.GetAvailableThings(state) {
		if !thing.HasPipe() {
			continue
		}

		if thing.Snapshot().Angular.Pipe.Name != name {
			continue
		}

		pipeIdent := tcb.GetPipeIdent(thing)
		if pipeIdent == "" {
			pipeIdent = "_pipe" + utils.GetNextStringId()

			impIdent := tcb.AddImport(thing)

			declStatement := Statement{}
			declStatement.AddVirtPart("const " + pipeIdent + " = null! as " + impIdent + "\n")
			tcb.AddPipeDeclaration(pipeIdent, thing, &declStatement)
		}

		statement.AddVirtPart(pipeIdent)
		statement.AddVirtPart(".")
		statement.AddRealPart("transform", node)
		statement.AddVirtPart("(")

		break
	}

	return &statement
}
