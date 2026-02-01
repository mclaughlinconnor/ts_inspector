package utils

import (
	"regexp"
	"strings"
)

func IsAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.-]+\])|(\([\w\.-]+\))|(\*\w)`, attribute)
}

func StripAngularFromAttribute(attribute string) string {
	if strings.HasPrefix(attribute, "[") && strings.HasSuffix(attribute, "]") {
		attribute = attribute[1 : len(attribute)-1]
	}

	if strings.HasPrefix(attribute, "(") && strings.HasSuffix(attribute, ")") {
		attribute = attribute[1 : len(attribute)-1]
	}

	return attribute
}
