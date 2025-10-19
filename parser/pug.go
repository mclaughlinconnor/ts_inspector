package parser

import (
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func HandlePugFile(state *State, class Class, uri string) (Class, error) {
	f, err := NewFile(uri, "pug", 0)
	if err != nil {
		return class, err
	}

	filename := f.Filename()
	state.Files[filename] = &f

	file := state.Files[filename]
	class.AngularTemplateFile = file

	class, err = utils.ParseFile(true, uri, utils.Pug, class, func(root *sitter.Node, content []byte, class Class) (Class, error) {
		file.SetContent(CStr2GoStr(content))

		class, err := ExtractPugUsages(class, content)

		if err != nil {
			return class, err
		}

		file.Classes = append(file.Classes, &class)

		return class, nil
	})

	return class, err
}

func ExtractPugUsages(file Class, content []byte) (Class, error) {
	pugFuncMap := walk.NewVisitorFuncsMap[Class]()
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
func extractIndentifierUsages(text []byte, class Class) (Class, error) {
	root, err := utils.GetRootNode(false, string(text), utils.JavaScript)
	if err != nil {
		return class, err
	}

	funcMap := walk.NewVisitorFuncsMap[Class]()
	funcMap["identifier"] = func(node *sitter.Node, state Class, indexInParent int, _ walk.VisitorFuncMap[Class]) Class {
		name := node.Content(text)
		usageInstance := UsageInstance{TemplateAccess, node}

		class = class.SetUsageAccessType(name, usageInstance.Access).AppendUsage(name, usageInstance)

		return class
	}

	class = walk.Walk(root, class, funcMap)

	return class, nil
}

// func assignTemplate(controller string, state State, template string) State {
// 	c, found := state.Classes[controller]
// 	if !found {
// 		return state
// 	}
//
// 	c.Template = template
// 	state.Classes[controller] = c
//
// 	return state
// }

// func assignController(template string, state State, controller string) State {
// 	c, found := state.Classes[template]
// 	if !found {
// 		return state
// 	}
//
// 	c.Controller = controller
// 	state.Classes[template] = c
//
// 	return state
// }

func visitAttribute(content []byte) walk.VisitorFunction[Class] {
	return func(node *sitter.Node, state Class, indexInParent int, _ walk.VisitorFuncMap[Class]) Class {
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
			} else if child.Type() == "javascript" {
				valueNode = child
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
		if valueNode.Type() == "javascript" && strings.HasPrefix(string(value), "`") && strings.HasSuffix(string(value), "`") {
			value = value[1 : len(value)-1]
		}
		state, _ = extractIndentifierUsages(value, state)

		return state
	}
}

func visitContent(content []byte) walk.VisitorFunction[Class] {
	return func(node *sitter.Node, state Class, indexInParent int, _ walk.VisitorFuncMap[Class]) Class {
		tagContent := []byte(node.Content(content))

		angularContentFuncMap := walk.NewVisitorFuncsMap[Class]()
		angularContentFuncMap["interpolation"] = func(node *sitter.Node, state Class, indexInParent int, _ walk.VisitorFuncMap[Class]) Class {
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
