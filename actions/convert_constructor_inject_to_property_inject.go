package actions

import (
	"io"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func ConvertInjectToProperty(_ io.Writer, state *parser.State, file *parser.File, r utils.Range) (actionEdits *utils.TextEdits, command *interfaces.Command, allowed bool, err error) {
	if file.Snapshot().Filetype != "typescript" {
		return nil, nil, false, nil
	}

	startByte := file.GetOffsetForPosition(r.Start)
	endByte := file.GetOffsetForPosition(r.End)

	action := actionEditHolder{[]utils.TextEdit{}, true}
	content := []byte(file.Snapshot().Content)
	root, err := utils.ParseText(content, utils.TypeScript)
	if err != nil {
		return retActionErr(err)
	}

	parameterNode := ast.HasNodeInHierarchy(root, "required_parameter", startByte, endByte)
	if parameterNode == nil {
		return retActionErr(err)
	}

	nameNode := parameterNode.ChildByFieldName("pattern")
	if nameNode == nil {
		return retActionErr(err)
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
		return retActionErr(err)
	}

	definition, found := class.Snapshot().Definitions[name]
	if !found {
		return retActionErr(err)
	}

	importEdits, err := ast.AddImportToFile(content, "@angular/core", []string{"inject"}, []string{})
	if err != nil {
		return retAction(action, err)
	}

	action.Edits = append(action.Edits, importEdits...)

	var text string
	score := 300

	if definition.IsLocalParam() {
		return retActionErr(err)
	} else if definition.IsPublic() {
		text = "  public "
		score = score + 2
	} else if definition.IsPrivate() {
		text = "  private "
	} else {
		return retActionErr(err)
	}

	if definition.Override {
		text = text + "override "
	}

	if definition.Readonly {
		text = text + "readonly "
	}

	hasIndexDecorator := definition.HasInjectDecorator()

	text = text + name + " = inject("

	if hasIndexDecorator {
		for _, decorator := range definition.Decorators {
			if decorator.Name != "Inject" {
				continue
			}

			text = text + strings.Join(decorator.Arguments, ", ")
		}
	} else {
		text += definition.Type
	}

	text = text + ");"

	propertyEdits, err := ast.AddMethodDefinitionToFile(content, text, name, score)
	if err != nil {
		return retAction(action, err)
	}

	action.Edits = append(action.Edits, propertyEdits...)

	startPosition := utils.GetPositionForOffset(file.Snapshot().Content, parameterNode.StartByte())

	endOffset := parameterNode.EndByte()
	sibling := parameterNode.NextSibling()
	if sibling != nil && sibling.Type() == "," {
		endOffset = endOffset + 1
	}

	sibling = sibling.NextNamedSibling()
	if sibling != nil && sibling.StartPoint().Row > parameterNode.EndPoint().Row {
		endOffset = sibling.StartByte()
	}
	endPosition := utils.GetPositionForOffset(file.Snapshot().Content, endOffset)

	rr := utils.Range{Start: startPosition, End: endPosition}
	removeParameter := utils.TextEdit{Range: rr, NewText: ""}
	action.Edits = append(action.Edits, removeParameter)

	return &action.Edits, nil, action.IsAllowed, err
}
