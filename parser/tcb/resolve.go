package tcb

import (
	"slices"
	"ts_inspector/parser"
)

func (t *Tag) ResolveSourceClassOfAttribute(state *parser.State, attribute *Attribute, currentClass *parser.Class) *parser.Class {
	if attribute.SourceClass != nil {
		return attribute.SourceClass
	}

	tagSourceClass := t.ResolveSourceClassOfTag(state, currentClass)

	for _, definition := range tagSourceClass.FilterAllDefinitions(func(def parser.ClassedDefinition) bool { return def.NameMatchesString(attribute.Name) }) {
		attribute.SourceClass = definition.Class
		return attribute.SourceClass
	}

	things := currentClass.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		if !thing.HasDirective() {
			continue
		}

		if slices.ContainsFunc(thing.Snapshot().Angular.Directive.Selectors, t.matchesSelector) {
			attribute.SourceClass = thing
			return attribute.SourceClass
		}
	}

	return nil
}

func (t *Tag) ResolveSourceClassOfTag(state *parser.State, currentClass *parser.Class) *parser.Class {
	if t.SourceClass != nil {
		return t.SourceClass
	}

	if !currentClass.HasComponent() {
		return nil
	}

	things := currentClass.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		if !thing.HasComponent() {
			continue
		}

		t.SourceClass = thing
		return t.SourceClass
	}

	return nil
}
