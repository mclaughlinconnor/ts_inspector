package parser

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTemplateFilename(controller File, state State, controllerStatePath string, root *sitter.Node, content []byte) (State, error) {
	return WithMatches(QueryComponentDecorator, TypeScript, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		relativeTemplatePath := captures[2].Node.Content(content)
		controllerDirectory := filepath.Dir(controllerStatePath)

		templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativeTemplatePath))
		if err != nil {
			return returnValue, err
		}

		if FileExists(templateFilePath) {
			returnValue = assignTemplate(controller.Filename(), returnValue, templateFilePath)
			returnValue = assignController(templateFilePath, returnValue, controller.Filename())
			return returnValue, nil
		}

		return returnValue, fmt.Errorf("Unexpected template state does not exist: %s", relativeTemplatePath)
	}))
}

func ExtractPugUsages(file File, state State, content []byte) (State, error) {
	state, err := WithMatches(QueryAttribute, Pug, content, state, func(captures []sitter.QueryCapture, returnValue State) (State, error) {

		name := []byte(captures[0].Node.Content(content))

		isAttr, err := isAngularAttribute(name)

		if err != nil {
			return returnValue, err
		}

		if isAttr {
			valueNode := captures[1].Node
			value := []byte(valueNode.Content(content))
			return extractIndentifierUsages(value, file, returnValue)
		}

		return returnValue, nil
	})

	if err != nil {
		return state, err
	}

	return WithMatches(QueryContent, Pug, content, state, HandleMatch[State](func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		tagContentNode := captures[0].Node
		tagContent := []byte(tagContentNode.Content(content))

		return WithMatches(QueryInterpolation, AngularContent, tagContent, state, func(captures []sitter.QueryCapture, returnValue State) (State, error) {
			interpolationNode := captures[0].Node
			interpolation := []byte(interpolationNode.Content(tagContent))

			return extractIndentifierUsages(interpolation, file, state)
		})
	}))
}

// Intentionally only get `identifier`s instead of `property_identifier`s because only the `identifier` will exist on the controller
func extractIndentifierUsages(text []byte, file File, state State) (State, error) {
	return WithMatches(QueryPropertyUsage, JavaScript, text, state, func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		node := captures[0].Node
		name := node.Content(text)
		usageInstance := UsageInstance{ForeignAccess, node}

		template := state[file.Filename()]

		template = template.SetUsageAccessType(name, usageInstance.Access)
		template = template.AppendUsage(name, usageInstance)
		state[template.Filename()] = template

		if file.Controller != "" {
			controller := state[file.Controller]
			controller = controller.AppendDefinitionUsage(name, usageInstance)
			state[controller.Filename()] = controller
		}

		return state, nil
	})

}

func isAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.]+\])|(\([\w\.]+\))|(\*\w)`, attribute)
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
