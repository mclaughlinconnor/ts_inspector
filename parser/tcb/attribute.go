package tcb

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/parser"
	structuraldirective "ts_inspector/parser/structural_directive"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Attribute struct {
	renderable
	tcb       *Tcb
	value     string
	valueExpr *structuraldirective.ShorthandValue

	Name      string // includes angular [] and ()
	NameNode  *sitter.Node
	Node      *sitter.Node
	Mixin     *Mixin
	Tag       *Tag
	ValueNode *sitter.Node
}

func (a *Attribute) GetExpression() (*structuraldirective.Expression, error) {
	if !a.IsStructuralInput() {
		return &structuraldirective.Expression{Expression: a.value}, nil
	}

	valueShv, err := a.GetShv()
	if err != nil {
		return nil, err
	}

	return valueShv.GetExpression(), nil
}

func (a *Attribute) GetShv() (*structuraldirective.ShorthandValue, error) {
	if a.valueExpr == nil {
		v, err := structuraldirective.ParseShorthand(a.GetStrippedName(), a.value)
		if err != nil {
			return nil, err
		}

		a.valueExpr = v
	}

	return a.valueExpr, nil
}

func (a *Attribute) GetSourceClass() *parser.Class {
	return a.Tcb().Class
}

func (a *Attribute) GetStrippedName() string {
	return utils.StripAngularFromAttributeNoType(a.Name)
}

func (a *Attribute) IsInput() bool {
	return a.IsStructuralInput() || strings.HasPrefix(a.Name, "[") && strings.HasSuffix(a.Name, "]")
}

func (a *Attribute) IsStructuralInput() bool {
	return strings.HasPrefix(a.Name, "*")
}

func (a *Attribute) IsOutput() bool {
	return strings.HasPrefix(a.Name, "[") && strings.HasSuffix(a.Name, "]")
}

func (a *Attribute) MatchesSelector(selector string, ignoreTag bool) (bool, *ast.Selector) {
	s, err := ast.ParseSelector(selector)
	if err != nil {
		return false, s
	}

	return a.MatchesParsedSelector(s, ignoreTag)
}

func (a *Attribute) MatchesParsedSelector(selector *ast.Selector, ignoreTag bool) (bool, *ast.Selector) {
	if !ignoreTag && selector.Tag != "" {
		if a.Tag.Name != selector.Tag {
			return false, selector
		}
	}

	if len(selector.Attributes) > 0 {
		found := false
		for _, selectorAttr := range selector.Attributes {
			if a.GetStrippedName() == selectorAttr {
				found = true
				break
			}
		}

		if !found {
			return false, selector
		}
	}

	if !ignoreTag && len(selector.NotTags) > 0 {
		if slices.Contains(selector.NotTags, a.Tag.Name) {
			return false, selector
		}
	}

	if len(selector.NotAttributes) > 0 {
		for _, selectorAttr := range selector.Attributes {
			if a.GetStrippedName() == selectorAttr {
				return false, selector
			}
		}
	}

	return true, selector
}

func (a *Attribute) SetSourceClass(class *parser.Class) {
	a.Tcb().Class = class
}

func (a *Attribute) Tcb() *Tcb {
	return a.tcb
}

func (t *Tag) renderAttributes() error {
	allAttributes := map[string]*Attribute{}
	for _, a := range t.Attributes.Elements {
		attributeName, _ := utils.StripAngularFromAttribute(a.Attribute.Name)
		allAttributes[attributeName] = a.Attribute
	}

	renderedDirectives := map[string]bool{}

	for _, a := range t.Attributes.Elements {
		rd, err := renderAttribute(&allAttributes, a.Attribute, renderedDirectives)
		if err != nil {
			return err
		}

		renderedDirectives = rd
	}

	return nil
}

func renderAttribute(allAttributes *map[string]*Attribute, attribute *Attribute, renderedDirectives map[string]bool) (map[string]bool, error) {
	if !attribute.IsInput() {
		return renderedDirectives, nil
	}

	sourceClass := attribute.Tcb().Class
	if !sourceClass.HasComponent() {
		return renderedDirectives, nil
	}

	tcb := attribute.Tcb()
	state := tcb.State
	component := sourceClass.Snapshot().Angular.Component

	things := component.GetAvailableThings(state)

	hasMatched := false

THING:
	for _, thing := range things {
		for _, selector := range thing.GetSelectors() {
			matchesSelector, _ := attribute.Tag.MatchesSelector(selector)
			if !matchesSelector {
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
				if isAttached {
					attachedInputs[inputName] = a
					continue
				}

				for attributeName, attribute := range *allAttributes {
					if !attribute.IsStructuralInput() {
						continue
					}

					if strings.HasPrefix(inputName, attributeName) {
						attachedInputs[inputName] = attribute
					}
				}
			}

			if len(attachedInputs) == 0 {
				continue THING
			}

			hasMatched = true
			renderedDirectives[thing.Id()] = true

			classIdent := tcb.AddImport(thing)
			declIdent := buildDirectiveDeclaration(tcb, thing)

			assIdent, err := buildDirectiveAssignment(tcb, thing, attribute, declIdent, &attachedInputs)
			if err != nil {
				return map[string]bool{}, err
			}

			ctxIdent, err := buildGuards(tcb, attribute, thing, assIdent, classIdent)
			if err != nil {
				return map[string]bool{}, err
			}

			if attribute.IsStructuralInput() {
				valueShv, err := attribute.GetShv()
				if err != nil {
					return map[string]bool{}, err
				}

				buildStructuralShorthandContextExpansion(attribute, tcb, valueShv, ctxIdent)
			}

			continue THING
		}
	}

	if !hasMatched {
		valueExpr, err := attribute.GetExpression()
		if err != nil {
			return map[string]bool{}, err
		}

		if valueExpr != nil {
			expr := buildTcbExpression(tcb.Ast, valueExpr.Expression)
			if attribute.ValueNode != nil {
				expr.OffsetByNodeStart(attribute.ValueNode)
			}

			tcb.AddVirtPart("(")
			tcb.AddStatement(expr)
			tcb.AddVirtPart(");\n")
		}
	}

	return renderedDirectives, nil
}

func buildDirectiveAssignment(tcb *Tcb, thing *parser.Class, attribute *Attribute, declIdent string, attachedInputs *map[string]*Attribute) (string, error) {
	assIdent := declIdent
	if len(thing.Snapshot().TypeParameters) > 0 {
		ai, err := buildGenericDirectiveAssignment(tcb, attribute, thing, declIdent, attachedInputs)
		if err != nil {
			return "", err
		}
		assIdent = ai
	}

	buildNonGenericDirectiveAssignment(tcb, attribute, thing, assIdent, attachedInputs)

	return assIdent, nil
}

func buildGuards(tcb *Tcb, attribute *Attribute, thing *parser.Class, assIdent string, classIdent string) (string, error) {
	if !thing.HasDirective() && !strings.HasPrefix(attribute.Name, "*") && attribute.Tag.Name != "ng-template" {
		return "", nil
	}

	hasContextGuard := false
	var inputGuard *parser.ClassedDefinition = nil

	strippedAttributeName := utils.StripAngularFromAttributeNoType(attribute.Name)
	inputGuardDefName := NG_TEMPLATE_GUARD_PREFIX + strippedAttributeName

	for _, definition := range thing.GetAllDefinitions() {
		if definition.Name == NG_TEMPLATE_CONTEXT_GUARD {
			hasContextGuard = true
		}

		if inputGuard != nil {
			if hasContextGuard {
				break
			} else {
				continue
			}
		}

		// For `@Input('alias') public prop`, `ngTemplateGuard_alias` and `ngTemplateGuard_prop` are both valid
		if strings.HasPrefix(definition.Name, NG_TEMPLATE_GUARD_PREFIX) {
			if definition.Name == NG_TEMPLATE_GUARD_PREFIX+strippedAttributeName {
				inputGuard = &definition
			}
		} else {
			inputName := definition.GetInputName()
			if inputName == strippedAttributeName {
				thing.GetDefinition(NG_TEMPLATE_GUARD_PREFIX + inputName)
			}
		}
	}

	if !hasContextGuard && inputGuard == nil {
		return "", nil
	}

	valueExpr, err := attribute.GetExpression()
	if err != nil {
		return "", err
	}

	var value *Statement
	if valueExpr != nil {
		value = buildTcbExpression(tcb.Ast, valueExpr.Expression)
		if attribute.ValueNode != nil {
			value.OffsetByNodeStart(attribute.ValueNode)
		}
	} else {
		value = StatementFromString(UNDEFINED)
	}

	statement := Statement{}
	statement.AddVirtPart("if (")

	ctxIdent := tcb.CreateVarInCurrentScope(StatementFromString(NULL_AS_ANY), classIdent)
	if hasContextGuard {
		statement.AddVirtPart(fmt.Sprintf("%s.%s(%s, %s)", classIdent, NG_TEMPLATE_CONTEXT_GUARD, assIdent, ctxIdent))
	}

	if hasContextGuard && inputGuard != nil {
		statement.AddVirtPart(" && ")
	}

	if inputGuard != nil {
		ttype := utils.StripQuotes(inputGuard.Type)
		if ttype == "binding" {
			statement.AddVirtPart("(")
			statement.AddStatement(value)
			statement.AddVirtPart(")")
		} else {
			statement.AddVirtPart(fmt.Sprintf("%s.%s(%s, ", classIdent, inputGuardDefName, assIdent))
			if !strings.HasPrefix(attribute.Name, "*") { // non-structural don't have an expression, it's just a binding
				statement.AddVirtPart(UNDEFINED)
			} else {
				statement.AddStatement(value)
			}
			statement.AddVirtPart(")")
		}
	}

	statement.AddVirtPart(")") // close the if

	tcb.AddStatement(&statement)

	if len(attribute.Tag.Children.Elements) != 0 {
		tcb.BeginRealScope(attribute.Tag.Children.Elements[0].Tag.Node)
	} else {
		tcb.BeginScope()
	}

	tcb.AddVirtPart("(" + ctxIdent + ");\n")
	if attribute.Tag.Identifier == "" {
		attribute.Tag.AddDeclaration(false, false)
	}

	attribute.Tag.closeScope = true

	return ctxIdent, nil
}

func buildGenericDirectiveAssignment(tcb *Tcb, attribute *Attribute, thing *parser.Class, compIdent string, attachedInputs *map[string]*Attribute) (string, error) {
	ctorExpr := Statement{}
	ctorExpr.AddVirtPart(compIdent)
	ctorExpr.AddVirtPart("({")

	values := map[string]*Statement{}

	for _, input := range thing.GetInputs(true) {
		inputName := input.GetInputName()
		attached, isAttached := (*attachedInputs)[inputName]
		if !isAttached {
			values[inputName] = nil
			continue
		}

		valueShv, err := attached.GetShv()
		if err != nil {
			return "", err
		}

		if !attached.IsStructuralInput() || inputName == valueShv.Prefix {
			if attached.ValueNode == nil {
				continue
			}

			valueExpr, err := attached.GetExpression()
			if err != nil {
				return "", err
			}

			if valueExpr == nil {
				values[inputName] = nil
				continue
			}

			v := buildTcbExpression(tcb.Ast, valueExpr.Expression)
			if attached.ValueNode != nil {
				v.OffsetByNodeStart(attached.ValueNode)
			}

			values[inputName] = v
		}

		hasKeyExp, keyExp := valueShv.GetKeyExprWithKey(inputName)
		if !hasKeyExp {
			// Don't overwrite the existing value
			if values[inputName] == nil {
				values[inputName] = nil
			}
			continue
		}

		v := buildTcbExpression(tcb.Ast, keyExp.Expression)
		if attached.ValueNode != nil {
			v.OffsetByNodeStart(attached.ValueNode)
		}

		values[inputName] = v
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

	assIdent := tcb.CreateVarInCurrentScope(&ctorExpr, "")

	return assIdent, nil
}

func buildNonGenericDirectiveAssignment(tcb *Tcb, attribute *Attribute, thing *parser.Class, dirIdent string, attachedInputs *map[string]*Attribute) error {
	nullAssignment := func(def *parser.ClassedDefinition) {
		tcb.AddAssignment(dirIdent+"."+def.Name, nil, StatementFromString(NULL_AS_ANY))
	}

	for _, def := range thing.GetInputs(true) {
		inputName := def.GetInputName()
		attached, isAttached := (*attachedInputs)[inputName]
		if !isAttached {
			continue
		}

		if attached.ValueNode == nil {
			nullAssignment(&def)
		}

		valueShv, err := attached.GetShv()
		if err != nil {
			return err
		}

		if !attached.IsStructuralInput() || inputName == valueShv.Prefix {
			if attached.ValueNode == nil {
				continue
			}

			valueExpr, err := attached.GetExpression()
			if err != nil {
				return err
			}

			if valueExpr == nil {
				nullAssignment(&def)
				continue
			}

			value := buildTcbExpression(tcb.Ast, valueExpr.Expression)
			value.OffsetByNodeStart(attached.ValueNode)
			tcb.AddAssignment(dirIdent+"."+def.Name, attached.NameNode, value)

			continue
		}

		hasKeyExp, keyExp := valueShv.GetKeyExprWithKey(inputName)
		if !hasKeyExp {
			continue
		}

		value := buildTcbExpression(tcb.Ast, keyExp.Expression)
		value.OffsetByNodeStart(attached.ValueNode).OffsetByOffset(keyExp.ExpressionOffset)
		tcb.AddAssignment(dirIdent+"."+def.Name, attached.NameNode, value)
	}

	valueShv, err := attribute.GetShv()
	if err != nil {
		return err
	}

	if valueShv == nil {
		return nil
	}

	return nil
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

	ident := tcb.CreateVarInRootScope(&value, "")

	tcb.AddDirectiveConstructor(ident, thing, nil, false)

	return ident
}

// The expansion for stuff that affects the context
func buildStructuralShorthandContextExpansion(attribute *Attribute, tcb *Tcb, shv *structuraldirective.ShorthandValue, ctxIdent string) {
	for _, statement := range shv.Statements.Elements {
		if statement.HasExpression() {
			expr := statement.Expression
			if expr.Local != nil {
				tcb.CreateVarInCurrentScope(StatementFromString(ctxIdent+"."+expr.Expression), *expr.Local)
			}

			continue
		}

		if statement.HasLet() {
			let := statement.Let

			var export string
			if let.Export != nil {
				export = *let.Export
			} else {
				export = IMPLICIT
			}

			text := ctxIdent + "." + export
			valueStatement := buildStatementFromAttributeAndOffset(attribute, text, export, let.ExportOffset)

			tcb.CreateVarInCurrentScope(valueStatement, let.Local)

			continue
		}

		// if statement.HasKeyExp() {} // doesn't affect context
	}
}

func buildStatementFromAttributeAndOffset(attribute *Attribute, tsText string, pugText string, shvOffset int) *Statement {
	valueStatement := Statement{}

	pugStartOffset := int(attribute.ValueNode.StartByte()) + shvOffset
	pugEndOffset := pugStartOffset + len(pugText)

	tsStartOffset := 0
	tsEndOffset := tsStartOffset + len(tsText)
	part := Part{
		node:           nil,
		text:           tsText,
		PugEndOffset:   &pugStartOffset,
		PugStartOffset: &pugEndOffset,
		TsEndOffset:    &tsEndOffset,
		TsStartOffset:  &tsStartOffset,
		Id:             getNextId(),
	}

	valueStatement.AddPartRaw(&part)

	return &valueStatement
}
