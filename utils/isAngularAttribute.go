package utils

import (
	"regexp"
	"strings"
)

const (
	NeitherAngularStripped = 0
	InputAngularStripped   = 1
	OutputAngularStripped  = 2
	BothAngularStripped    = 3
)

func IsAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.-]+\])|(\([\w\.-]+\))|(\*\w)`, attribute)
}

func StripAngularFromAttribute(attribute string) (string, int) {
	mode := NeitherAngularStripped

	if strings.HasPrefix(attribute, "[") && strings.HasSuffix(attribute, "]") {
		attribute = attribute[1 : len(attribute)-1]
		mode |= InputAngularStripped
	}

	if strings.HasPrefix(attribute, "(") && strings.HasSuffix(attribute, ")") {
		attribute = attribute[1 : len(attribute)-1]
		mode |= OutputAngularStripped
	}

	return attribute, mode
}
