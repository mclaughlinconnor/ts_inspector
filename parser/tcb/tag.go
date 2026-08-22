package tcb

import (
	"slices"
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Tag struct {
	tcb        *Tcb
	closeScope bool

	Attributes   utils.HelpfulArray[*Node]
	Children     utils.HelpfulArray[*Node]
	Content      utils.HelpfulArray[*TagContent]
	Identifier   string
	Name         string
	NameNode     *sitter.Node
	Node         *sitter.Node
	SourceClass  *parser.Class
	TemplateRefs utils.HelpfulArray[TemplateRef]
}

type TagContent struct {
	Interpolation string
	Node          *sitter.Node
	Text          string
}

type TagContentArray struct {
	//nolint:unused
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

	t.Attributes.Add(node)

	return node
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
			matchesSelector, _ := t.MatchesSelector(selector)
			if !matchesSelector {
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

func (t *Tag) HasAttribute(attribute string) bool {
	strippedAttr, _ := utils.StripAngularFromAttribute(attribute)
	return slices.ContainsFunc(t.Attributes.Elements, func(a *Node) bool {
		attr := a.Attribute
		if attr.GetStrippedName() == strippedAttr {
			return true
		}

		if attr.IsStructuralInput() {
			shv, err := attr.GetShv()
			if err != nil {
				return false
			}

			return shv.GetKeyExprWithKey(strippedAttr) != nil
		} else {
			sa, _ := utils.StripAngularFromAttribute(attr.Name)
			return sa == strippedAttr
		}
	})
}

func (t *Tag) HasAttributes(attributes []string) bool {
	for _, attribute := range attributes {
		if !t.HasAttribute(attribute) {
			return false
		}
	}

	return true
}

func (t *Tag) MatchesParsedSelector(selector *ast.Selector) (bool, *ast.Selector) {
	if selector.Tag != "" {
		if t.Name != selector.Tag {
			return false, selector
		}
	}

	if len(selector.Attributes) > 0 {
		if !t.HasAttributes(selector.Attributes) {
			return false, selector
		}
	}

	if len(selector.NotTags) > 0 {
		if slices.Contains(selector.NotTags, t.Name) {
			return false, selector
		}
	}

	if len(selector.NotAttributes) > 0 {
		if !t.NotHasAttributes(selector.NotAttributes) {
			return false, selector
		}
	}

	return true, selector
}

func (t *Tag) MatchesSelector(selector string) (bool, *ast.Selector) {
	s, err := ast.ParseSelector(selector)
	if err != nil {
		return false, s
	}

	return t.MatchesParsedSelector(s)
}

func (t *Tag) NotHasAttributes(attributes []string) bool {
	return !slices.ContainsFunc(attributes, t.HasAttribute)
}

func (t *Tag) Render() error {
	tcb := t.Tcb()

	if len(tcb.CurrentScope.Parts.Parts) > 0 {
		tcb.TagBoundaryPartStack.Push(tcb.CurrentScope.Parts.Parts[len(tcb.CurrentScope.Parts.Parts)-1])
	}

	err := t.renderAttributes()
	if err != nil {
		return err
	}

	if t.Identifier == "" {
		t.AddDeclaration(true, false)
	}

	for _, c := range t.Children.Elements {
		err := c.Render()
		if err != nil {
			return err
		}
	}

	if t.closeScope {
		tcb.EndScope()
	}

	tcb.TagBoundaryPartStack.Pop()

	return nil
}

func (t *Tag) Tcb() *Tcb {
	return t.tcb
}

// Spaghetti
func (t *Tag) insertValue(value *Statement, insertBeforeCurrentTag bool, shouldAddReferenceVar bool) {
	tcb := t.Tcb()
	if insertBeforeCurrentTag {
		ident, newAfter := tcb.CreateVarAfterPart(value, "", *tcb.TagBoundaryPartStack.Peek())
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
		refIdent, _ := tcb.CreateVarAfterPart(value, "", newAfter)
		for i := range t.TemplateRefs.Elements {
			t.TemplateRefs.Elements[i].Identifier = refIdent
		}
	} else {
		t.Identifier = tcb.CreateVarInCurrentScope(value, "")
		if shouldAddReferenceVar {
			value = StatementFromString(t.Identifier)
			tcb.CreateVarInCurrentScope(value, "")
		}
	}
}

func findTemplateRefClass(tcb *Tcb) *parser.Class {
	for _, class := range tcb.State.GetClasses() {
		if class.Snapshot().Name != "TemplateRef" {
			continue
		}

		return class
	}

	return nil
}
