package parser

import (
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

func IndexPugFileFromLsp(state *State, uri string, content string, version int) error {
	filetype, err := FiletypeFromFilename(FilenameFromUri(uri))
	if err != nil || filetype != "pug" {
		return err
	}

	file, err := createFileIfNotExists(state, FilenameFromUri(uri), content, version)
	if err != nil {
		return err
	}

	file.ResetDeclarations()
	indexPug(state, file)

	return nil
}
func IndexPugFromIndexer(state *State, templateFileName string) error {
	filetype, err := FiletypeFromFilename(templateFileName)
	if err != nil || filetype != "pug" {
		return err
	}

	file, err := createFileIfNotExists(state, templateFileName, "", 0)
	if err != nil {
		return err
	}

	file.ResetDeclarations()

	return indexPug(state, file)
}

func IndexPugFromTypeScript(state *State, class *Class, templateFileName string) error {
	filetype, err := FiletypeFromFilename(templateFileName)
	if err != nil || filetype != "pug" {
		return err
	}

	file, err := createFileIfNotExists(state, templateFileName, "", 0)
	if err != nil {
		return err
	}

	file.ResetDeclarations()
	class.EnsureAngular()
	class.Snapshot().Angular.EnsureComponent()
	class.Snapshot().Angular.Component.TemplateUrlFile = file

	err = extractPugUsages(class, []byte(file.Snapshot().Content))
	if err != nil {
		return err
	}

	return nil
}

// Intentionally only get `identifier`s instead of `property_identifier`s because only the `identifier` will exist on the controller
func extractIndentifierUsages(text []byte, class *Class) error {
	root, err := utils.GetRootNode(false, string(text), utils.JavaScript)
	if err != nil {
		return err
	}

	funcMap := walk.NewVisitorFuncsMap[*Class]()
	funcMap["identifier"] = func(node *sitter.Node, state *Class, indexInParent int, _ walk.VisitorFuncMap[*Class]) *Class {
		name := node.Content(text)

		usageInstance := UsageInstance{Access: TemplateAccess, Class: state, Node: node}

		class.SetUsageAccessType(name, usageInstance.Access)
		class.AppendUsage(name, &usageInstance)
		class.AppendDefinitionUsage(name, &usageInstance)

		return class
	}
	class = walk.WalkJavaScript(root, class, funcMap)

	return nil
}

func extractPugUsages(class *Class, content []byte) error {
	pugFuncMap := walk.NewVisitorFuncsMap[*Class]()
	pugFuncMap["attribute"] = visitAttribute(content)
	pugFuncMap["content"] = visitContent(content)
	pugFuncMap["tag_name"] = func(node *sitter.Node, state *Class, indexInParent int, _ walk.VisitorFuncMap[*Class]) *Class {
		if state.Snapshot().Angular == nil {
			logger.Printf("Somehow class.Angular has ended up nil. I have no idea how. Class: %v\n", state.Snapshot().Name)
			return state
		}

		if state.Snapshot().Angular.Component == nil {
			logger.Printf("Somehow class.Angular.Component has ended up nil. I have no idea how. Class: %v\n", state.Snapshot().Name)
			return state
		}

		state.Snapshot().Angular.Component.AddTagUsage(node, node.Content(content))

		return state
	}

	root, err := utils.GetRootNode(false, string(content), utils.Pug)
	if err != nil {
		return err
	}

	output := walk.WalkPug(root, class, pugFuncMap)
	if output != class {
		panic("ExtractPugUsages altered the class pointer")
	}

	return nil
}

func indexPug(state *State, file *File) error {
	for _, class := range *state.GetClasses() {
		if class.GetTemplateFile() == file {
			class.DropTemplateUsages()

			err := extractPugUsages(class, []byte(file.Snapshot().Content))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func visitAttribute(content []byte) walk.VisitorFunction[*Class] {
	return func(node *sitter.Node, state *Class, indexInParent int, _ walk.VisitorFuncMap[*Class]) *Class {
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

		extractIndentifierUsages(value, state)

		return state
	}
}

func visitContent(content []byte) walk.VisitorFunction[*Class] {
	return func(node *sitter.Node, state *Class, indexInParent int, _ walk.VisitorFuncMap[*Class]) *Class {
		tagContent := []byte(node.Content(content))

		angularContentFuncMap := walk.NewVisitorFuncsMap[*Class]()
		angularContentFuncMap["interpolation"] = func(node *sitter.Node, state *Class, indexInParent int, _ walk.VisitorFuncMap[*Class]) *Class {
			interpolation := []byte(node.Content(tagContent))
			_ = extractIndentifierUsages(interpolation, state)
			return state
		}

		angularRoot, err := utils.GetRootNode(false, string(tagContent), utils.AngularContent)
		if err != nil {
			return state
		}

		state = walk.WalkAngular(angularRoot, state, angularContentFuncMap)
		return state
	}
}
