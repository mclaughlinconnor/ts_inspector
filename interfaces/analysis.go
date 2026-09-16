package interfaces

import "ts_inspector/utils"

type Analysis struct {
	Code string

	Message string

	Range utils.Range

	RelatedInformation []RelatedInformation

	Severity int

	Source string
}

type RelatedInformation struct {
	Message string
	Uri     string
	Range   utils.Range
}

type severity struct {
	Error int

	Warning int

	Information int

	Hint int
}

var AnalysisSeverity = severity{1, 2, 3, 4}

type Category int32

func (category Category) Name() string {
	switch category {
	case CategoryWarning:
		return "warning"
	case CategoryError:
		return "error"
	case CategorySuggestion:
		return "suggestion"
	case CategoryMessage:
		return "message"
	}
	panic("Unhandled diagnostic category")
}

const (
	CategoryWarning Category = iota
	CategoryError
	CategorySuggestion
	CategoryMessage
)

func AnalysisSeverityFromTsGoCategory(category *Category) int {
	switch *category {
	case CategoryWarning:
		return AnalysisSeverity.Warning
	case CategoryError:
		return AnalysisSeverity.Error
	case CategorySuggestion:
		return AnalysisSeverity.Hint
	case CategoryMessage:
		return AnalysisSeverity.Information
	default:
		return AnalysisSeverity.Error
	}
}

func ErrorToAnalyses(err error) Analysis {
	return NewAnalysis("analysisError", utils.ZeroRange(), AnalysisSeverity.Error, err.Error(), nil)
}

func ErrorsToAnalyses(errors []error) []Analysis {
	if len(errors) == 0 {
		return []Analysis{}
	}

	analyses := []Analysis{}
	for _, err := range errors {
		analyses = append(analyses, ErrorToAnalyses(err))
	}

	return analyses
}

func NewAnalysis(code string, highlightRange utils.Range, severity int, message string, relatedInformation *[]RelatedInformation) Analysis {
	var ri []RelatedInformation
	if relatedInformation == nil {
		ri = []RelatedInformation{}
	} else {
		ri = *relatedInformation
	}

	return Analysis{Code: code, Message: message, Range: highlightRange, RelatedInformation: ri, Severity: severity, Source: "ts_inspector"}
}
