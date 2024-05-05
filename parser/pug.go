package parser

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTemplateFilename(controllerFilePath string, root *sitter.Node, content []byte) (filename string, err error) {
	return WithMatches(QueryComponentDecorator, TypeScript, content, "", HandleMatch[string](func(captures []sitter.QueryCapture, returnValue string) (string, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		relativeTemplatePath := captures[2].Node.Content(content)
		controllerDirectory := filepath.Dir(controllerFilePath)

		templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativeTemplatePath))
		if err != nil {
			return "", err
		}

		if FileExists(templateFilePath) {
			return templateFilePath, nil
		}

		return "", fmt.Errorf("Expected template file does not exist: %s", templateFilePath)
	}))
}

func ExtractPugUsages(state State, content []byte) (State, error) {
	state, err := WithMatches(QueryAttribute, Pug, content, state, func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		name := []byte(captures[0].Node.Content(content))

		isAttr, err := isAngularAttribute(name)

		if err != nil {
			return state, err
		}

		if isAttr {
			valueNode := captures[1].Node
			value := []byte(valueNode.Content(content))
			return extractIndentifierUsages(value, state)
		}

		return state, nil
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

			return extractIndentifierUsages(interpolation, state)
		})
	}))
}

// Intentionally only get `identifier`s instead of `property_identifier`s because only the `identifier` will exist on the controller
func extractIndentifierUsages(text []byte, state State) (State, error) {
	return WithMatches(QueryPropertyUsage, JavaScript, text, state, func(captures []sitter.QueryCapture, returnValue State) (State, error) {
		node := captures[0].Node
		name := node.Content(text)
		usageInstance := UsageInstance{ForeignAccess, node}

		usage, ok := state.Usages[name]
		if ok {
			existingUsages := usage
			existingUsages.Access = CalculateNewAccessType(existingUsages.Access, usageInstance.Access)
			existingUsages.Usages = append(existingUsages.Usages, usageInstance)
			state.Usages[name] = existingUsages
		} else {
			state.Usages[name] = Usage{
				usageInstance.Access,
				name,
				[]UsageInstance{usageInstance},
			}
		}

		return state, nil
	})

}

func isAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.]+\])|(\([\w\.]+\))|(\*\w)`, attribute)
}
