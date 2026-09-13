package utils

import "strconv"

var ids = map[string]int{"global": 0}

func GetNextIdGlobal() int {
	return GetNextId("global")
}

func GetNextId(id string) int {
	next := ids[id]
	ids[id] = next + 1

	return next
}

func GetNextStringIdGlobal() string {
	return strconv.Itoa(GetNextIdGlobal())
}

func GetNextStringId(id string) string {
	return strconv.Itoa(GetNextId(id))
}

func ResetNextId(id string) {
	ids[id] = 0
}
