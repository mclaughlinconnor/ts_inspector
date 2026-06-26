package indexing

import (
	"io/fs"
	"path/filepath"
)

var projectRootWalking = false

func Index(rootUri string) ([]string, []string) {
	roots, tsconfigFiles := findProjectRoots(rootUri)

	files := []string{}

	if projectRootWalking == true {
		for _, root := range roots {
			i, e := recursivelyRetrieveImports(root, 0, 1)
			if e == nil {
				files = append(files, i...)
			}
		}
	}

	filepath.WalkDir(rootUri, func(path string, d fs.DirEntry, err error) error {
		if !d.Type().IsDir() {
			ext := filepath.Ext(path)
			if ext != ".ts" && ext != ".pug" {
				return nil
			}

			files = append(files, path)
			return nil
		}

		if d.Name() != "node_modules" {
			return nil
		}

		return fs.SkipDir
	})

	return files, tsconfigFiles
}
