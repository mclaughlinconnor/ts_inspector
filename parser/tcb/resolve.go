package tcb

import (
	"ts_inspector/parser"
)

func (t *Tag) ResolveSourceClassOfAttribute(state *parser.State, attribute *Attribute, currentClass *parser.Class) *parser.Class {
	if attribute.GetSourceClass() != nil {
		return attribute.GetSourceClass()
	}

	tagSourceClass := t.ResolveSourceClassOfTag(state, currentClass)
	if tagSourceClass == nil {
		return nil
	}

	for _, definition := range tagSourceClass.FilterAllDefinitions(func(def parser.ClassedDefinition) bool { return def.NameMatchesString(attribute.Name) }) {
		attribute.SetSourceClass(definition.Class)
		return attribute.GetSourceClass()
	}

	things := currentClass.Snapshot().Angular.Component.GetAvailableThings(state)
	for _, thing := range things {
		if !thing.HasDirective() {
			continue
		}

		matches := false
		for _, selector := range thing.Snapshot().Angular.Directive.Selectors {
			match, _ := t.MatchesSelector(selector)
			if !match {
				continue
			}

			matches = true
		}

		if matches {
			attribute.SetSourceClass(thing)
			return attribute.GetSourceClass()
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
