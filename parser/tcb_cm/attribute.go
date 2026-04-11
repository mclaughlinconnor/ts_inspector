package tcb_cm

import (
	"strings"
	"ts_inspector/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

type Attribute struct {
	renderable
	tcb *Tcb

	Name      string // includes angular [] and ()
	NameNode  *sitter.Node
	Node      *sitter.Node
	Tag       *Tag
	Value     string
	ValueNode *sitter.Node
}

func (a *Attribute) GetSourceClass() *parser.Class {
	return a.Tcb().Class
}

func (a *Attribute) IsInput() bool {
	return strings.HasPrefix(a.Name, "[") && strings.HasSuffix(a.Name, "]")
}

func (a *Attribute) IsOutput() bool {
	return strings.HasPrefix(a.Name, "[") && strings.HasSuffix(a.Name, "]")
}

func (a *Attribute) Render() {
	sourceClass := a.Tcb().Class
	if !sourceClass.HasComponent() {
		return
	}

	tcb := a.Tcb()
	state := tcb.State
	component := sourceClass.Snapshot().Angular.Component

	things := component.GetAvailableThings(state)

	for _, thing := range things {
		for _, selector := range thing.GetSelectors() {
			if !a.Tag.matchesSelector(selector) {
				continue
			}

			for _, def := range thing.GetAllDefinitions() {
				classIdent := tcb.AddImport(thing)

				value := StatementParts{}
				value.AddVirtPart("null! as " + classIdent)

				compIdent := tcb.CreateVar(value)

				if def.NameMatchesString(a.Name) {
					assInput := compIdent + "." + def.Name

					attrValue := a.Value
					if attrValue == "" {
						attrValue = UNDEFINED
					}

					valueExpr := buildTcbExpression(tcb, attrValue)
					if a.ValueNode != nil {
						valueExpr.OffsetByNodeStart(a.ValueNode)
					}

					tcb.AddAssignment(assInput, a.NameNode, *valueExpr)
				}
			}
		}
	}
}

func (a *Attribute) SetSourceClass(class *parser.Class) {
	a.Tcb().Class = class
}

func (a *Attribute) Tcb() *Tcb {
	return a.tcb
}
