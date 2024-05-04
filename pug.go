package main

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTemplateFilename(controllerFilePath string, root *sitter.Node, content []byte) (filename string, err error) {
	return WithCaptures(QueryComponentDecorator, TypeScript, content, "", HandleCapture[string](func(captures []sitter.QueryCapture, returnValue string) (string, error) {
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

func ExtractPugUsages(usages Usages, content []byte) (returnedUsages Usages, err error) {
	usages, err = WithCaptures(QueryAttribute, Pug, content, usages, func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
		name := []byte(captures[0].Node.Content(content))

		isAttr, err := isAngularAttribute(name)

		if err != nil {
			return usages, err
		}

		if isAttr {
			valueNode := captures[1].Node
			value := []byte(valueNode.Content(content))
			return extractIndentifierUsages(value, usages)
		}

		return usages, nil
	})

	return WithCaptures(QueryContent, Pug, content, usages, HandleCapture[Usages](func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
		tagContentNode := captures[0].Node
		tagContent := []byte(tagContentNode.Content(content))

		return WithCaptures(QueryInterpolation, AngularContent, tagContent, usages, func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
			interpolationNode := captures[0].Node
			interpolation := []byte(interpolationNode.Content(tagContent))

			return extractIndentifierUsages(interpolation, usages)
		})
	}))
}

// Intentionally only get `identifier`s instead of `property_identifier`s because only the `identifier` will exist on the controller
func extractIndentifierUsages(text []byte, usages Usages) (Usages, error) {
	return WithCaptures(QueryPropertyUsage, JavaScript, text, usages, func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
		node := captures[0].Node
		name := node.Content(text)
		usageInstance := UsageInstance{ForeignAccess, node}

		_, ok := usages[name]
		if ok {
			existingUsages := usages[name]
			existingUsages.Access = CalculateNewAccessType(existingUsages.Access, usageInstance.Access)
			existingUsages.Usages = append(existingUsages.Usages, usageInstance)
			usages[name] = existingUsages
		} else {
			usages[name] = Usage{
				usageInstance.Access,
				name,
				[]UsageInstance{usageInstance},
			}
		}

		return usages, nil
	})

}

func isAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.]+\])|(\([\w\.]+\))|(\*\w)`, attribute)
}
