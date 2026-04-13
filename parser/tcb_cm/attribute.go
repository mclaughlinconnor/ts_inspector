package tcb_cm

import (
	"fmt"
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

THING:
	for _, thing := range things {
		for _, selector := range thing.GetSelectors() {
			if !a.Tag.matchesSelector(selector) {
				continue
			}

			for _, def := range thing.GetAllDefinitions() {
				classIdent := tcb.AddImport(thing)

				if def.NameMatchesString(a.Name) {
					value := StatementParts{}
					value.AddVirtPart("null! as " + classIdent)

					compIdent := tcb.CreateVar(value)

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

					if strings.HasPrefix(a.Name, "*") && thing.HasDirective() && len(thing.FilterAllDefinitions(func(d parser.ClassedDefinition) bool { return d.Name == parser.NG_TEMPLATE_CONTEXT_GUARD })) > 0 {
						ctxIdent := tcb.CreateVar(*StatementPartsFromString(NULL_AS_ANY))

						tcb.AddVirtPart(fmt.Sprintf("if (%s.%s(%s, %s)) {\n", classIdent, parser.NG_TEMPLATE_CONTEXT_GUARD, compIdent, ctxIdent))

						a.Tag.postParts = StatementPartsFromString("\n}\n")
					}
				}
			}

			continue THING
		}
	}
}

func (a *Attribute) SetSourceClass(class *parser.Class) {
	a.Tcb().Class = class
}

func (a *Attribute) Tcb() *Tcb {
	return a.tcb
}
