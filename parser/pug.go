package parser

import (
	"regexp"
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
	file, err := utils.WithMatches(utils.QueryAttribute, utils.Pug, content, file, func(captures utils.Captures, returnValue File) (File, error) {

		name := []byte(captures["name"][0].Content(content))

		isAttr, err := isAngularAttribute(name)

		if err != nil {
			return returnValue, err
		}

		if isAttr {
			valueNode := captures["value"][0]
			value := []byte(valueNode.Content(content))
			return extractIndentifierUsages(value, returnValue)
		}

		return returnValue, nil
	})

	if err != nil {
		return file, err
	}

	return utils.WithMatches(utils.QueryContent, utils.Pug, content, file, func(captures utils.Captures, returnValue File) (File, error) {
		tagContentNode := captures["content"][0]
		tagContent := []byte(tagContentNode.Content(content))

		return utils.WithMatches(utils.QueryInterpolation, utils.AngularContent, tagContent, file, func(captures utils.Captures, returnValue File) (File, error) {
			interpolationNode := captures["interpolation"][0]
			interpolation := []byte(interpolationNode.Content(tagContent))

			return extractIndentifierUsages(interpolation, returnValue)
		})
	})
}

// Intentionally only get `identifier`s instead of `property_identifier`s because only the `identifier` will exist on the controller
func extractIndentifierUsages(text []byte, file File) (File, error) {
	return utils.WithMatches(utils.QueryPropertyUsage, utils.JavaScript, text, file, func(captures utils.Captures, returnValue File) (File, error) {
		node := captures["name"][0]
		name := node.Content(text)
		usageInstance := UsageInstance{ForeignAccess, node}

		file = file.SetUsageAccessType(name, usageInstance.Access).AppendUsage(name, usageInstance)

		return file, nil
	})

}

func isAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.-]+\])|(\([\w\.-]+\))|(\*\w)`, attribute)
}

func assignTemplate(controller string, state State, template string) State {
	f, found := state[controller]
	if !found {
		return state
	}

	f.Template = template
	state[controller] = f

	return state
}

func assignController(template string, state State, controller string) State {
	f, found := state[template]
	if !found {
		return state
	}

	f.Controller = controller
	state[template] = f

	return state
}
