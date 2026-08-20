package utils

import (
	"path"

	"golang.org/x/sync/syncmap"
)

var cache = syncmap.Map{}

func PathDir(fullpath string) string {
	dir, found := cache.Load(fullpath)
	if found {
		return dir.(string)
	}

	dir = path.Dir(fullpath)
	cache.Store(fullpath, dir)

	return dir.(string)
}
