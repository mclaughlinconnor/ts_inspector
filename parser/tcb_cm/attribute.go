package tcb_cm

import (
	"strings"
	"ts_inspector/parser"

	sitter "github.com/smacker/go-tree-sitter"
)

type Attribute struct {
	renderable
	tcb *Tcb

	Name  string // includes angular [] and ()
	Node  *sitter.Node
	Tag   *Tag
	Value string
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
				if def.NameMatchesString(a.Name) {
					classIdent := tcb.AddImport(thing)

					value := StatementParts{}
					value.AddPart("null! as " + classIdent)

					compIdent := tcb.CreateVar(value)

					assInput := compIdent + "." + def.Name

					attrValue := a.Value
					if attrValue == "" {
						attrValue = UNDEFINED
					}

					tcb.AddAssignment(assInput, StatementParts{[]string{buildTcbExpression(attrValue)}})

					continue THING
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
