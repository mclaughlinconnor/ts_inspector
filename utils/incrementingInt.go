package utils

import "strconv"

var nextId = 1

func GetNextId() int {
	id := nextId
	nextId++

	return id
}

func GetNextStringId() string {
	return strconv.Itoa(GetNextId())
}
