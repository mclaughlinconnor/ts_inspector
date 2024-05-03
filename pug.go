package main

import (
	"fmt"
	"path"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTemplateFilename(controllerFilePath string, root *sitter.Node, content []byte) (filename string, err error) {
	return WithCaptures(QueryComponentDecorator, TypeScript, content, HandleCapture[string](func(captures []sitter.QueryCapture, returnValue string) (string, error) {
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
	return WithCaptures(QueryContent, Pug, content, HandleCapture[Usages](func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
		tagContentNode := captures[0].Node
		tagContent := []byte(tagContentNode.Content(content))

		return WithCaptures(QueryInterpolation, AngularContent, tagContent, func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
			interpolationNode := captures[0].Node
			interpolation := []byte(interpolationNode.Content(tagContent))

			return WithCaptures(QueryPropertyUsage, JavaScript, interpolation, func(captures []sitter.QueryCapture, returnValue Usages) (Usages, error) {
				usageInstance := UsageInstance{ForeignAccess, tagContentNode}
				name := captures[0].Node.Content(interpolation)

				_, ok := usages[name]
				if ok {
					existingUsages := usages[name]
					existingUsages.Usages = append(existingUsages.Usages, usageInstance)
					existingUsages.Access = ForeignAccess // Pug usages are always foreign
					usages[name] = existingUsages
				} else {
					usages[name] = Usage{
						LocalAccess,
						name,
						[]UsageInstance{usageInstance},
					}
				}

				return usages, nil
			})
		})
	}))
}
