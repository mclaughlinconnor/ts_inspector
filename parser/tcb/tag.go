package tcb

import (
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Tag struct {
	tcb        *Tcb
	closeScope bool

	Attributes   HelpfulArray[*Node]
	Children     HelpfulArray[*Node]
	Content      HelpfulArray[*TagContent]
	Identifier   string
	Name         string
	NameNode     *sitter.Node
	Node         *sitter.Node
	SourceClass  *parser.Class
	TemplateRefs HelpfulArray[TemplateRef]
}

type TagContent struct {
	Interpolation string
	Node          *sitter.Node
	Text          string
}

type TagContentArray struct {
	elems []*TagContent
}

type TemplateRef struct {
	Attribute  *Node
	Identifier string
	Name       string
	Tag        *Tag
	Value      string
}

func (t *Tag) addAttribute(attribute *Attribute) *Node {
	node := newAttributeNode(attribute)

	t.Attributes.add(node)

	return node
}

func (t *Tag) matchesSelector(selector string) bool {
	if t.Name == selector {
		return true
	}

	valid, tagName, attrName := ast.ExtractTagNameAndAttrFromSelector(selector)
	if !valid || (tagName != "" && t.Name != tagName) {
		return false
	}

	for _, attr := range t.Attributes.Elements {
		attr := attr.Attribute.Name

		if attr == attrName {
			return true
		}

		angularlessAttr, _ := utils.StripAngularFromAttribute(attr)
		if angularlessAttr == attrName {
			return true
		}
	}

	return false
}

func (t *Tag) AddDeclaration(insertBeforeCurrentTag bool, shouldAddReferenceVar bool) {
	tcb := t.Tcb()

	if t.Name == "ng-template" {
		t.insertValue(StatementFromString("null! as any"), insertBeforeCurrentTag, true)
		return
	}

	things := tcb.Class.Snapshot().Angular.Component.GetAvailableThings(t.tcb.State)
	for _, thing := range things {
		if !thing.HasComponent() {
			continue
		}

		for _, selector := range thing.Snapshot().Angular.Component.Selectors {
			if !t.matchesSelector(selector) {
				continue
			}

			classIdent := StatementFromString("null! as " + tcb.AddImport(thing))
			typeParameters := thing.Snapshot().TypeParameters
			if len(typeParameters) > 0 {
				classIdent.AddVirtPart("<")
				for i := range typeParameters {
					if i > 0 {
						classIdent.AddVirtPart(", ")
					}

					classIdent.AddVirtPart("any")
				}
				classIdent.AddVirtPart(">")
			}

			t.insertValue(classIdent, insertBeforeCurrentTag, shouldAddReferenceVar)

			return
		}
	}

	t.insertValue(StatementFromString("document.createElement(\""+t.Name+"\")"), insertBeforeCurrentTag, shouldAddReferenceVar)
}

func (t *Tag) Render() {
	tcb := t.Tcb()

	if len(tcb.CurrentScope.Parts.Parts) > 0 {
		tcb.TagBoundaryPartStack.Push(tcb.CurrentScope.Parts.Parts[len(tcb.CurrentScope.Parts.Parts)-1])
	}

	t.renderAttributes()

	if t.Identifier == "" {
		t.AddDeclaration(true, false)
	}

	for _, c := range t.Children.Elements {
		c.Render()
	}

	if t.closeScope {
		tcb.EndScope()
	}

	tcb.TagBoundaryPartStack.Pop()
}

func (t *Tag) Tcb() *Tcb {
	return t.tcb
}

// Spaghetti
func (t *Tag) insertValue(value *Statement, insertBeforeCurrentTag bool, shouldAddReferenceVar bool) {
	tcb := t.Tcb()
	if insertBeforeCurrentTag {
		ident, newAfter := tcb.CreateVarAfterPart(value, *tcb.TagBoundaryPartStack.Peek())
		t.Identifier = ident

		if !shouldAddReferenceVar {
			return
		}

		var value *Statement
		if t.Name == "ng-template" {
			if len(t.TemplateRefs.Elements) == 0 {
				// This one ends up in the wrong place. Compare the output of `ng-template([ngIf]='true', [ngIfElse]='id')`
				value = StatementFromString(t.Identifier)
			} else {
				trClass := findTemplateRefClass(tcb)
				if trClass != nil {
					trIdent := t.tcb.AddImport(trClass)
					value = StatementFromString("(" + t.Identifier + " as any as " + trIdent + "<any>)")
				} else { // shouldn't happen
					value = StatementFromString("(" + t.Identifier + " as any)")
				}
			}
		} else {
			value = StatementFromString(t.Identifier)
		}
		refIdent, _ := tcb.CreateVarAfterPart(value, newAfter)
		for i := range t.TemplateRefs.Elements {
			t.TemplateRefs.Elements[i].Identifier = refIdent
		}
	} else {
		t.Identifier = tcb.CreateVarInCurrentScope(value)
		if shouldAddReferenceVar {
			value = StatementFromString(t.Identifier)
			tcb.CreateVarInCurrentScope(StatementFromString(t.Identifier))
		}
	}
}

func findTemplateRefClass(tcb *Tcb) *parser.Class {
	for _, class := range *tcb.State.GetClasses() {
		if class.Snapshot().Name != "TemplateRef" {
			continue
		}

		return class
	}

	return nil
}
