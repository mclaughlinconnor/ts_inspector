package parser

import (
	"os"
	"path"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
)

type Reference struct {
	Class *Class
	Name  string
	Node  *sitter.Node

	// Variable *Variable
	// ...
}

type References []*Reference

func (r *References) IterateResolved(yield func(*Reference) bool) {
	for _, v := range *r {
		if v == nil {
			continue
		}
		if !yield(v) {
			return
		}
	}
}

func resolveIdentFromImports(idents []string, file *File, state *State) []*Reference {
	resolved := make([]*Reference, len(idents))

	for identIndex, ident := range idents {
		importPath := file.FindImportPath(ident)

		if importPath == "" {
			continue
		}

		var importedFile *File

		extensions := []string{".ts", ".d.ts", ".js"}
		joinSuffixes := []string{"", "index"}

		for _, join := range joinSuffixes {
			ip := path.Join(importPath, join)
			for _, extension := range extensions {
				importedFile = resolveProjectImportPath(state, file, ip+extension)
				if importedFile != nil {
					break
				}
			}

			if importedFile == nil {
				for _, extension := range extensions {
					importedFile = resolveNodeModulesImportPath(state, file, ip+extension)
					if importedFile != nil {
						break
					}
				}
			}

			if importedFile != nil {
				break
			}
		}

		for _, export := range importedFile.Exports {
			if export.Name == ident {
				resolved[identIndex] = export
			}
		}
	}

	return resolved
}

func resolveNodeModulesImportPath(state *State, currentFile *File, importPath string) *File {
	currentPath := filepath.Dir(FilenameFromUri(currentFile.URI))

	for currentPath != "." && currentPath != "/" {
		nmPath := path.Join(currentPath, "node_modules")
		stat, err := os.Stat(nmPath)
		if err != nil {
			currentPath = path.Dir(currentPath)
			continue
		}

		if stat.IsDir() {
			resolvedFile := getFileByPath(state, path.Join(nmPath, importPath))
			if resolvedFile != nil {
				return resolvedFile
			}
		}

		currentPath = path.Dir(currentPath)
	}

	resolvedFile := getFileByPath(state, currentPath)
	if resolvedFile != nil {
		return resolvedFile
	}

	return nil
}
