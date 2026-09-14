package utils

import (
	"strconv"
	"sync"
)

type Ids struct {
	sync.Mutex
	ids map[string]int
}

var ids = Ids{sync.Mutex{}, map[string]int{"global": 0}}

func GetNextId(id string) int {
	ids.Lock()
	defer ids.Unlock()

	next := ids.ids[id]
	ids.ids[id] = next + 1

	return next
}

func GetNextIdGlobal() int {
	return GetNextId("global")
}

func GetNextStringId(id string) string {
	return strconv.Itoa(GetNextId(id))
}

func GetNextStringIdGlobal() string {
	return strconv.Itoa(GetNextIdGlobal())
}

func ResetNextId(id string) {
	ids.Lock()
	defer ids.Unlock()

	ids.ids[id] = 0
}
