package utils

import "strings"

func StripQuotes(text string) string {
	hasDoubleQuote := strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"")
	hasSingleQuote := strings.HasPrefix(text, "'") && strings.HasSuffix(text, "'")
	if !hasDoubleQuote && !hasSingleQuote {
		return text
	}

	quote := "'"
	if hasDoubleQuote {
		quote = "\""
	}

	return strings.TrimPrefix(strings.TrimSuffix(text, quote), quote)

}
