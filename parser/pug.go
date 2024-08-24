package parser

import (
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func HandlePugFile(file File) (File, error) {
	fromDisk := file.Content == ""
	var source string
	if fromDisk {
		source = file.Filename()
	} else {
		source = file.Content
	}

	return utils.ParseFile(fromDisk, source, utils.Pug, file, func(root *sitter.Node, content []byte, file File) (File, error) {
		file = file.SetContent(CStr2GoStr(content))

		file, err := ExtractPugUsages(file, content)
		if err != nil {
			return file, err
		}
		return file, nil
	})
}

func ExtractPugUsages(file File, content []byte) (File, error) {
	pugFuncMap := walk.NewVisitorFuncsMap[File]()
	pugFuncMap["attribute"] = visitAttribute(content)
	pugFuncMap["content"] = visitContent(content)

	root, err := utils.GetRootNode(false, string(content), utils.Pug)
	if err != nil {
		return file, err
	}

	file = walk.Walk(root, file, pugFuncMap)

	return file, nil
}

// Intentionally only get `identifier`s instead of `property_identifier`s because only the `identifier` will exist on the controller
func extractIndentifierUsages(text []byte, file File) (File, error) {
	root, err := utils.GetRootNode(false, string(text), utils.JavaScript)
	if err != nil {
		return file, err
	}

	funcMap := walk.NewVisitorFuncsMap[File]()
	funcMap["identifier"] = func(node *sitter.Node, state File, indexInParent int, _ walk.VisitorFuncMap[File]) File {
		name := node.Content(text)
		usageInstance := UsageInstance{ForeignAccess, node}

		file = file.SetUsageAccessType(name, usageInstance.Access).AppendUsage(name, usageInstance)

		return file
	}

	file = walk.Walk(root, file, funcMap)

	return file, nil
}

func assignTemplate(controller string, state State, template string) State {
	f, found := state.Files[controller]
	if !found {
		return state
	}

	f.Template = template
	state.Files[controller] = f

	return state
}

func assignController(template string, state State, controller string) State {
	f, found := state.Files[template]
	if !found {
		return state
	}

	f.Controller = controller
	state.Files[template] = f

	return state
}

func visitAttribute(content []byte) walk.VisitorFunction[File] {
	return func(node *sitter.Node, state File, indexInParent int, _ walk.VisitorFuncMap[File]) File {
		var nameNode *sitter.Node
		var valueNode *sitter.Node

		for childIndex := range node.NamedChildCount() {
			child := node.NamedChild(int(childIndex))
			if child.Type() == "attribute_name" {
				nameNode = child
			} else if child.Type() == "quoted_attribute_value" {
				v := child.NamedChild(0)
				if v != nil && v.Type() == "attribute_value" {
					valueNode = v
				}
			}
		}

		if nameNode == nil || valueNode == nil {
			return state
		}

		attrName := nameNode.Content(content)
		isAttr, err := utils.IsAngularAttribute([]byte(attrName))

		if err != nil || !isAttr {
			return state
		}

		value := []byte(valueNode.Content(content))
		state, _ = extractIndentifierUsages(value, state)

		return state
	}
}

func visitContent(content []byte) walk.VisitorFunction[File] {
	return func(node *sitter.Node, state File, indexInParent int, _ walk.VisitorFuncMap[File]) File {
		tagContent := []byte(node.Content(content))

		angularContentFuncMap := walk.NewVisitorFuncsMap[File]()
		angularContentFuncMap["interpolation"] = func(node *sitter.Node, state File, indexInParent int, _ walk.VisitorFuncMap[File]) File {
			interpolation := []byte(node.Content(tagContent))
			state, _ = extractIndentifierUsages(interpolation, state)
			return state
		}

		angularRoot, err := utils.GetRootNode(false, string(tagContent), utils.AngularContent)
		if err != nil {
			return state
		}

		state = walk.Walk(angularRoot, state, angularContentFuncMap)
		return state
	}
}
