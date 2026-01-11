package actions

import (
	"io"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/parser"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func ConvertInjectToProperty(_ io.Writer, state *parser.State, file *parser.File, r utils.Range) (actionEdits utils.TextEdits, allowed bool, err error) {
	if file.Snapshot().Filetype != "typescript" {
		return nil, false, nil
	}

	startByte := file.GetOffsetForPosition(r.Start)
	endByte := file.GetOffsetForPosition(r.End)

	action := addDestroyedAction{[]utils.TextEdit{}, true}
	action, err = utils.ParseFile(false, file.Snapshot().Content, utils.TypeScript, action, func(root *sitter.Node, content []byte, edits addDestroyedAction) (addDestroyedAction, error) {
		parameterNode := ast.HasNodeInHierarchy(root, "required_parameter", startByte, endByte)
		if parameterNode == nil {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		}

		nameNode := parameterNode.ChildByFieldName("pattern")
		if nameNode == nil {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		}

		name := nameNode.Content(content)

		classes := file.Snapshot().Classes
		var class *parser.Class
		for _, c := range classes {
			node := c.Snapshot().Node
			if node.StartByte() <= startByte && node.EndByte() >= endByte {
				class = c
			}
		}

		if class == nil {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		}

		definition, found := class.Snapshot().Definitions[name]
		if !found {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		}

		if !definition.HasInjectDecorator() {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		}

		importEdits, err := ast.AddImportToFile(content, "@angular/core", []string{"inject"}, []string{})
		if err != nil {
			return action, err
		}

		action.Edits = append(action.Edits, importEdits...)

		var text string
		score := 300

		if definition.IsLocalParam() {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		} else if definition.IsPublic() {
			text = "  public "
			score = score + 2
		} else if definition.IsPrivate() {
			text = "  private "
		} else {
			return addDestroyedAction{[]utils.TextEdit{}, false}, err
		}

		if definition.Override {
			text = text + "override "
		}

		if definition.Readonly {
			text = text + "readonly "
		}

		text = text + name + " = inject("

		for _, decorator := range definition.Decorators {
			if decorator.Name != "Inject" {
				continue
			}

			text = text + strings.Join(decorator.Arguments, ", ")
		}

		text = text + ");"

		propertyEdits, err := ast.AddMethodDefinitionToFile(content, text, name, score)
		if err != nil {
			return edits, err
		}

		action.Edits = append(action.Edits, propertyEdits...)

		startPosition := parser.GetPositionForOffset(file.Snapshot().Content, parameterNode.StartByte())

		endOffset := parameterNode.EndByte()
		sibling := parameterNode.NextSibling()
		if sibling != nil && sibling.Type() == "," {
			endOffset = endOffset + 1
		}

		sibling = sibling.NextNamedSibling()
		if sibling != nil && sibling.StartPoint().Row > parameterNode.EndPoint().Row {
			endOffset = sibling.StartByte()
		}
		endPosition := parser.GetPositionForOffset(file.Snapshot().Content, endOffset)

		r := utils.Range{Start: startPosition, End: endPosition}

		removeParameter := utils.TextEdit{Range: r, NewText: ""}
		action.Edits = append(action.Edits, removeParameter)

		return action, nil
	})

	return action.Edits, action.IsAllowed, err
}
