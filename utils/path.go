package utils

import "path"

var cache = map[string]string{}

func PathDir(fullpath string) string {
	dir, found := cache[fullpath]
	if found {
		return dir
	}

	dir = path.Dir(fullpath)
	cache[fullpath] = dir

	return dir
}
