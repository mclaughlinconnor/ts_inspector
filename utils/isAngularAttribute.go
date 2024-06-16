package utils

import "regexp"

func IsAngularAttribute(attribute []byte) (bool, error) {
	return regexp.Match(`(\[[\w\.-]+\])|(\([\w\.-]+\))|(\*\w)`, attribute)
}
