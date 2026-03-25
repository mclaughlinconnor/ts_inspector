package tcb

import "ts_inspector/parser"

type TmplDirectiveMetadata struct {
	directive       *parser.Directive
	isHostDirective bool
	// inputs map[string]Angular2DirectiveProperty,
	// outputs: Map<String Angular2DirectiveProperty>,
	// exportAs: Set<String>
}

func (t *TmplDirectiveMetadata) typeScriptClass(state *parser.State) *parser.Class {
	for _, class := range *state.GetClasses() {
		if !class.HasDirective() {
			continue
		}

		if class.Snapshot().Angular.Directive == t.directive {
			return class
		}
	}

	return nil
}

// func (t *TmplDirectiveMetadata) entityJsType() *JSType {
// 	if t.typeScriptClass && t.typeScriptClass.possiblyGenericJsType {
// 		return t.typeScriptClass.possiblyGenericJsType
// 	}
//
// 	return t.directive.entityJsType
// }

func (t *TmplDirectiveMetadata) isGeneric(state *parser.State) bool {
	c := t.typeScriptClass(state)

	return c != nil && len(c.Snapshot().TypeParameters) > 0
}

// func (t *TmplDirectiveMetadata) templateGuards() []Angular2TemplateGuard {
// 	return t.directive.templateGuards
// }

func (t *TmplDirectiveMetadata) hasTemplateContextGuard(state *parser.State) bool {
	c := t.typeScriptClass(state)
	if c == nil {
		return false
	}

	return len(c.FilterAllDefinitions(func(def parser.ClassedDefinition) bool { return def.NameMatchesString(NG_TEMPLATE_CONTEXT_GUARD) })) > 0
}
