package analysis

import (
	"fmt"
	"ts_inspector/parser"
)

func nonPublicAngular(_ *parser.State, file *parser.File) ([]Analysis, error) {
	return analyseClasses(file, func(class *parser.Class) ([]Analysis, error) {
		analyses := []Analysis{}

		for _, definition := range class.Snapshot().Definitions.All() {
			if definition.HasAngularDecorator() && !definition.IsPublic() && !definition.IsLocalParam() {
				message := fmt.Sprintf("Angular property should be public: %s", definition.Name)
				analyses = append(analyses, newAnalysisHighlightName(definition.Node, class, AnalysisSeverity.Warning, "non-public-angular", message))
			}
		}

		return analyses, nil
	}), nil
}
