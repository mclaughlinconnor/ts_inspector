package ast

import (
	"slices"
	"ts_inspector/parser"
)

func (t *Tag) ResolveSourceClassOfAttribute(state *parser.State, attribute *Attribute, currentClass *parser.Class) *parser.Class {
	tagSourceClass := t.ResolveSourceClassOfTag(state, currentClass)

	for _, definition := range tagSourceClass.FilterAllDefinitions(func(def parser.ClassedDefinition) bool { return def.NameMatchesString(attribute.Name) }) {
		return definition.Class
	}

	things := currentClass.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		if !thing.HasDirective() {
			continue
		}

		if slices.ContainsFunc(thing.Snapshot().Angular.Directive.Selectors, t.matchesSelector) {
			return thing
		}
	}

	return nil
}

func (t *Tag) ResolveSourceClassOfTag(state *parser.State, currentClass *parser.Class) *parser.Class {
	if !currentClass.HasComponent() {
		return nil
	}

	things := currentClass.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		if !thing.HasComponent() {
			continue
		}

		return thing
	}

	return nil
}
