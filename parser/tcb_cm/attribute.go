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

					compIdent := buildDirectiveDeclaration(tcb, thing)

					assInput := compIdent + "." + def.Name

					attrValue := a.Value
					if attrValue == "" {
						attrValue = UNDEFINED
					}

					valueExpr := buildTcbExpression(tcb, attrValue)
					if a.ValueNode != nil {
						valueExpr.OffsetByNodeStart(a.ValueNode)
					}

					dirIdent := buildDirectiveAssignment(tcb, thing, a, compIdent, assInput, &def, valueExpr)

					if strings.HasPrefix(a.Name, "*") && thing.HasDirective() && len(thing.FilterAllDefinitions(func(d parser.ClassedDefinition) bool { return d.Name == parser.NG_TEMPLATE_CONTEXT_GUARD })) > 0 {
						ctxIdent := tcb.CreateVar(*StatementPartsFromString(NULL_AS_ANY))

						tcb.AddVirtPart(fmt.Sprintf("if (%s.%s(%s, %s)) {\n", classIdent, parser.NG_TEMPLATE_CONTEXT_GUARD, dirIdent, ctxIdent))

						a.Tag.postParts = StatementPartsFromString("\n}\n")
					}
				}
			}

			continue THING
		}
	}
}

func buildDirectiveAssignment(tcb *Tcb, thing *parser.Class, attribute *Attribute, compIdent string, assInput string, def *parser.ClassedDefinition, value *StatementParts) string {
	if len(thing.Snapshot().TypeParameters) > 0 {
		return buildGenericDirectiveAssignment(tcb, thing, compIdent, def, value)
	}

	return buildNonGenericDirectiveAssignment(tcb, attribute, assInput, value)
}

func buildGenericDirectiveAssignment(tcb *Tcb, thing *parser.Class, compIdent string, def *parser.ClassedDefinition, value *StatementParts) string {
	ctorExpr := StatementParts{}
	ctorExpr.AddVirtPart(compIdent)
	ctorExpr.AddVirtPart("({")

	values := map[string]*StatementParts{}

	for _, input := range thing.GetInputs() {
		values[input.GetInputName()] = nil
	}

	values[def.GetInputName()] = value

	i := 0
	for k, v := range values {
		if i > 0 {
			ctorExpr.AddVirtPart(", ")
		}

		ctorExpr.AddVirtPart("\"" + k + "\": ")
		if v != nil {
			ctorExpr.AddStatementParts(v)
		} else {
			ctorExpr.AddVirtPart(NULL_AS_ANY)
		}

		i++
	}

	ctorExpr.AddVirtPart("})")

	return tcb.CreateVar(ctorExpr)
}

func buildNonGenericDirectiveAssignment(tcb *Tcb, attribute *Attribute, assInput string, value *StatementParts) string {
	tcb.AddAssignment(assInput, attribute.NameNode, *value)

	return assInput
}

func buildDirectiveDeclaration(tcb *Tcb, thing *parser.Class) string {
	if len(thing.Snapshot().TypeParameters) > 0 {
		return buildGenericDirectiveDeclaration(tcb, thing)
	}

	return buildNonGenericDirectiveDeclaration(tcb, thing)
}

func buildGenericDirectiveDeclaration(tcb *Tcb, thing *parser.Class) string {
	ctorIdent := "_ctor" + tcb.GetNextIdString()

	statement := StatementParts{}
	statement.AddVirtPart("const " + ctorIdent + ": ")

	tpValues := StatementParts{}
	tpDefs := StatementParts{}

	typeParameters := thing.Snapshot().TypeParameters
	if len(typeParameters) > 0 {
		tpDefs.AddVirtPart("<")
		tpValues.AddVirtPart("<")
		for i, tp := range typeParameters {
			if i > 0 {
				tpDefs.AddVirtPart(", ")
				tpValues.AddVirtPart(", ")
			}

			tpDefs.AddVirtPart(tp)
			tpValues.AddVirtPart(tp + " = any")
		}
		tpDefs.AddVirtPart(">")
		tpValues.AddVirtPart(">")
	}

	statement.AddStatementParts(&tpValues)

	thingIdent := tcb.AddImport(thing)
	thingDef := StatementParts{}
	thingDef.AddVirtPart(thingIdent)
	thingDef.AddStatementParts(&tpDefs)

	statement.AddVirtPart("(init: Pick<")
	statement.AddStatementParts(&thingDef)
	statement.AddVirtPart(", ")
	for i, input := range thing.GetInputs() {
		if i > 0 {
			statement.AddVirtPart(" | ")
		}

		statement.AddVirtPart("\"")
		statement.AddVirtPart(input.Name)
		statement.AddVirtPart("\"")
	}

	statement.AddVirtPart(">) => ")
	statement.AddStatementParts(&thingDef)
	statement.AddVirtPart(" = null!;\n")

	tcb.AddStatementParts(&statement)

	return ctorIdent
}

// var _t1 = null! as _i1.MacyDirectiveAgain;
func buildNonGenericDirectiveDeclaration(tcb *Tcb, thing *parser.Class) string {
	classIdent := tcb.AddImport(thing)
	value := StatementParts{}
	value.AddVirtPart("null! as " + classIdent)

	return tcb.CreateVar(value)
}

func (a *Attribute) SetSourceClass(class *parser.Class) {
	a.Tcb().Class = class
}

func (a *Attribute) Tcb() *Tcb {
	return a.tcb
}
