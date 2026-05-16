package tcb

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"ts_inspector/parser"
	"ts_inspector/utils"

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

func (a *Attribute) SetSourceClass(class *parser.Class) {
	a.Tcb().Class = class
}

func (a *Attribute) Tcb() *Tcb {
	return a.tcb
}

func (t *Tag) renderAttributes() {
	allAttributes := map[string]*Attribute{}
	for _, a := range t.Attributes.Elements {
		attributeName, _ := utils.StripAngularFromAttribute(a.Attribute.Name)
		allAttributes[attributeName] = a.Attribute
	}

	renderedDirectives := map[string]bool{}

	for _, a := range t.Attributes.Elements {
		renderedDirectives = renderAttribute(&allAttributes, a.Attribute, renderedDirectives)
	}
}

func renderAttribute(allAttributes *map[string]*Attribute, attribute *Attribute, renderedDirectives map[string]bool) map[string]bool {
	if !attribute.IsInput() {
		return renderedDirectives
	}

	sourceClass := attribute.Tcb().Class
	if !sourceClass.HasComponent() {
		return renderedDirectives
	}

	tcb := attribute.Tcb()
	state := tcb.State
	component := sourceClass.Snapshot().Angular.Component

	things := component.GetAvailableThings(state)

	hasMatched := false

THING:
	for _, thing := range things {
		for _, selector := range thing.GetSelectors() {
			if !attribute.Tag.matchesSelector(selector) {
				continue
			}

			if renderedDirectives[thing.Id()] {
				hasMatched = true
				continue THING
			}

			attachedInputs := map[string]*Attribute{}
			for _, def := range thing.GetAllDefinitions() {
				inputName := def.GetInputName()
				a, isAttached := (*allAttributes)[inputName]
				if !isAttached {
					continue
				}

				attachedInputs[inputName] = a
			}

			if len(attachedInputs) == 0 {
				continue THING
			}

			hasMatched = true
			renderedDirectives[thing.Id()] = true

			classIdent := tcb.AddImport(thing)
			declIdent := buildDirectiveDeclaration(tcb, thing)

			assIdent := buildDirectiveAssignment(tcb, thing, attribute, declIdent, &attachedInputs)

			if strings.HasPrefix(attribute.Name, "*") && thing.HasDirective() && len(thing.FilterAllDefinitions(func(d parser.ClassedDefinition) bool { return d.Name == parser.NG_TEMPLATE_CONTEXT_GUARD })) > 0 {
				ctxIdent := tcb.CreateVarInCurrentScope(StatementFromString(NULL_AS_ANY))

				tcb.AddVirtPart(fmt.Sprintf("if (%s.%s(%s, %s))", classIdent, parser.NG_TEMPLATE_CONTEXT_GUARD, assIdent, ctxIdent))
				tcb.BeginScope()
				tcb.AddVirtPart("(" + ctxIdent + ");\n")
				if attribute.Tag.Identifier == "" {
					attribute.Tag.AddDeclaration(false, false)
				}

				attribute.Tag.closeScope = true
			}

			continue THING
		}
	}

	if !hasMatched {
		expr := buildTcbExpression(tcb.Ast, attribute.Value)
		if attribute.ValueNode != nil {
			expr.OffsetByNodeStart(attribute.ValueNode)
		}

		tcb.AddVirtPart("(")
		tcb.AddStatement(expr)
		tcb.AddVirtPart(");\n")
	}

	return renderedDirectives
}

func buildDirectiveAssignment(tcb *Tcb, thing *parser.Class, attribute *Attribute, declIdent string, attachedInputs *map[string]*Attribute) string {
	assIdent := declIdent
	if len(thing.Snapshot().TypeParameters) > 0 {
		assIdent = buildGenericDirectiveAssignment(tcb, attribute, thing, declIdent, attachedInputs)
	}

	buildNonGenericDirectiveAssignment(tcb, attribute, thing, assIdent, attachedInputs)

	return assIdent
}

func buildGenericDirectiveAssignment(tcb *Tcb, attribute *Attribute, thing *parser.Class, compIdent string, attachedInputs *map[string]*Attribute) string {
	ctorExpr := Statement{}
	ctorExpr.AddVirtPart(compIdent)
	ctorExpr.AddVirtPart("({")

	values := map[string]*Statement{}

	for _, input := range thing.GetInputs(true) {
		inputName := input.GetInputName()
		attached, isAttached := (*attachedInputs)[inputName]
		if isAttached {
			v := buildTcbExpression(tcb.Ast, attached.Value)
			if attached.ValueNode != nil {
				v.OffsetByNodeStart(attached.ValueNode)
			}
			values[inputName] = v
		} else {
			values[inputName] = nil
		}
	}

	keys := slices.Collect(maps.Keys(values))
	slices.Sort(keys)

	for i, k := range keys {
		if i > 0 {
			ctorExpr.AddVirtPart(", ")
		}

		v := values[k]

		ctorExpr.AddVirtPart("\"" + k + "\": ")
		if v != nil {
			for _, p := range v.Parts {
				ctorExpr.AddVirtPart(p.text)
			}
		} else {
			ctorExpr.AddVirtPart(NULL_AS_ANY)
		}
	}

	ctorExpr.AddVirtPart("})")

	assIdent := tcb.CreateVarInCurrentScope(&ctorExpr)

	return assIdent
}

func buildNonGenericDirectiveAssignment(tcb *Tcb, attribute *Attribute, thing *parser.Class, dirIdent string, attachedInputs *map[string]*Attribute) {
	for _, def := range thing.GetInputs(true) {
		inputName := def.GetInputName()
		attached, isAttached := (*attachedInputs)[inputName]
		if !isAttached {
			continue
		}

		expr := buildTcbExpression(tcb.Ast, attached.Value)
		if attached.ValueNode != nil {
			expr.OffsetByNodeStart(attached.ValueNode)
		}

		tcb.AddAssignment(dirIdent+"."+def.Name, attached.NameNode, *expr)
	}
}

func buildDirectiveDeclaration(tcb *Tcb, thing *parser.Class) string {
	resolvedIdent := tcb.GetDirectiveIdent(thing)
	if resolvedIdent != "" {
		return resolvedIdent
	}

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
	for i, input := range thing.GetInputs(true) {
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

	tcb.AddDirectiveConstructor(ctorIdent, thing, &statement, true)

	return ctorIdent
}

// var _t1 = null! as _i1.MacyDirectiveAgain;
func buildNonGenericDirectiveDeclaration(tcb *Tcb, thing *parser.Class) string {
	classIdent := tcb.AddImport(thing)
	value := Statement{}
	value.AddVirtPart("null! as " + classIdent)

	ident := tcb.CreateVarInRootScope(&value)

	tcb.AddDirectiveConstructor(ident, thing, nil, false)

	return ident
}
