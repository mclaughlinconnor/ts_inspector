package utils

import "os"

// Is a var for replacing in tests
var FileExists = func(filename string) bool {
	stat, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}
