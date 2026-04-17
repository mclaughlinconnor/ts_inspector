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
	return (strings.HasPrefix(a.Name, "*")) || (strings.HasPrefix(a.Name, "[") && strings.HasSuffix(a.Name, "]"))
}

func (a *Attribute) IsOutput() bool {
	return strings.HasPrefix(a.Name, "[") && strings.HasSuffix(a.Name, "]")
}

func (a *Attribute) Render() {
	if !a.IsInput() {
		return
	}

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
					value := Statement{}
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
						ctxIdent := tcb.CreateVar(*StatementFromString(NULL_AS_ANY))

						tcb.AddVirtPart(fmt.Sprintf("if (%s.%s(%s, %s))", classIdent, parser.NG_TEMPLATE_CONTEXT_GUARD, dirIdent, ctxIdent))
						tcb.BeginScope()

						a.Tag.closeScope = true
					}
				}
			}

			continue THING
		}
	}
}

func buildDirectiveAssignment(tcb *Tcb, thing *parser.Class, attribute *Attribute, compIdent string, assInput string, def *parser.ClassedDefinition, value *Statement) string {
	if len(thing.Snapshot().TypeParameters) > 0 {
		return buildGenericDirectiveAssignment(tcb, thing, compIdent, def, value)
	}

	return buildNonGenericDirectiveAssignment(tcb, attribute, assInput, value)
}

func buildGenericDirectiveAssignment(tcb *Tcb, thing *parser.Class, compIdent string, def *parser.ClassedDefinition, value *Statement) string {
	ctorExpr := Statement{}
	ctorExpr.AddVirtPart(compIdent)
	ctorExpr.AddVirtPart("({")

	values := map[string]*Statement{}

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
			ctorExpr.AddStatement(v)
		} else {
			ctorExpr.AddVirtPart(NULL_AS_ANY)
		}

		i++
	}

	ctorExpr.AddVirtPart("})")

	return tcb.CreateVar(ctorExpr)
}

func buildNonGenericDirectiveAssignment(tcb *Tcb, attribute *Attribute, assInput string, value *Statement) string {
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

	statement := Statement{}
	statement.AddVirtPart("const " + ctorIdent + ": ")

	tpValues := Statement{}
	tpDefs := Statement{}

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

	statement.AddStatement(&tpValues)

	thingIdent := tcb.AddImport(thing)
	thingDef := Statement{}
	thingDef.AddVirtPart(thingIdent)
	thingDef.AddStatement(&tpDefs)

	statement.AddVirtPart("(init: Pick<")
	statement.AddStatement(&thingDef)
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
	statement.AddStatement(&thingDef)
	statement.AddVirtPart(" = null!;\n")

	tcb.AddStatement(&statement)

	return ctorIdent
}

// var _t1 = null! as _i1.MacyDirectiveAgain;
func buildNonGenericDirectiveDeclaration(tcb *Tcb, thing *parser.Class) string {
	classIdent := tcb.AddImport(thing)
	value := Statement{}
	value.AddVirtPart("null! as " + classIdent)

	return tcb.CreateVar(value)
}

func (a *Attribute) SetSourceClass(class *parser.Class) {
	a.Tcb().Class = class
}

func (a *Attribute) Tcb() *Tcb {
	return a.tcb
}
