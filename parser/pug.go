package parser

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"

	sitter "github.com/smacker/go-tree-sitter"
)

func ExtractTemplateFilename(file File, controllerFilePath string, root *sitter.Node, content []byte) (File, error) {
	return WithMatches(QueryComponentDecorator, TypeScript, content, file, HandleMatch[File](func(captures []sitter.QueryCapture, returnValue File) (File, error) {
		if len(captures) == 0 {
			return returnValue, nil
		}

		relativeTemplatePath := captures[2].Node.Content(content)
		controllerDirectory := filepath.Dir(controllerFilePath)

		templateFilePath, err := filepath.Abs(path.Join(controllerDirectory, relativeTemplatePath))
		if err != nil {
			return file, err
		}

		if FileExists(templateFilePath) {
			returnValue.Template = templateFilePath
			return returnValue, nil
		}

		return file, fmt.Errorf("Unexpected template file does not exist: %s %s %s", relativeTemplatePath, controllerFilePath, templateFilePath)
	}))
}

func ExtractPugUsages(file File, content []byte) (File, error) {
	file, err := WithMatches(QueryAttribute, Pug, content, file, func(captures []sitter.QueryCapture, returnValue File) (File, error) {
		name := []byte(captures[0].Node.Content(content))

		isAttr, err := isAngularAttribute(name)

		if err != nil {
			return file, err
		}

		if isAttr {
			valueNode := captures[1].Node
			value := []byte(valueNode.Content(content))
			return extractIndentifierUsages(value, file)
		}

		return file, nil
	})

	if err != nil {
		return file, err
	}

	return WithMatches(QueryContent, Pug, content, file, HandleMatch[File](func(captures []sitter.QueryCapture, returnValue File) (File, error) {
		tagContentNode := captures[0].Node
		tagContent := []byte(tagContentNode.Content(content))

		return WithMatches(QueryInterpolation, AngularContent, tagContent, file, func(captures []sitter.QueryCapture, returnValue File) (File, error) {
			interpolationNode := captures[0].Node
			interpolation := []byte(interpolationNode.Content(tagContent))

			return extractIndentifierUsages(interpolation, file)
		})
	}))
}

// Intentionally only get `identifier`s instead of `property_identifier`s because only the `identifier` will exist on the controller
func extractIndentifierUsages(text []byte, file File) (File, error) {
	return WithMatches(QueryPropertyUsage, JavaScript, text, file, func(captures []sitter.QueryCapture, returnValue File) (File, error) {
		node := captures[0].Node
		name := node.Content(text)
		usageInstance := UsageInstance{ForeignAccess, node}

		usage, ok := file.Usages[name]
		if ok {
			existingUsages := usage
			existingUsages.Access = CalculateNewAccessType(existingUsages.Access, usageInstance.Access)
			existingUsages.Usages = append(existingUsages.Usages, usageInstance)
			file.Usages[name] = existingUsages
		} else {
			file.Usages[name] = Usage{
				usageInstance.Access,
				name,
				[]UsageInstance{usageInstance},
			}
		}

		return file, nil
	})

}

func isAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.]+\])|(\([\w\.]+\))|(\*\w)`, attribute)
}
