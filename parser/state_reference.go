package parser

import (
	"os"
	"path"
	"slices"
	"sync"
	"ts_inspector/config"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Reference struct {
	Class    *Class
	File     *File
	Name     string
	Node     *sitter.Node
	Variable *Variable
}

type References []*Reference

func (r References) ContainsClass(class *Class) bool {
	return slices.ContainsFunc(r, func(r *Reference) bool {
		if r == nil || r.Class == nil {
			return false
		}

		return r.Class == class
	})
}

func (r References) CountResolved() int {
	count := 0

	for _, v := range r {
		if v == nil {
			continue
		}

		count += 1
	}

	return count
}

func (r References) IterateResolved(yield func(*Reference) bool) {
	for _, v := range r {
		if v == nil {
			continue
		}
		if !yield(v) {
			return
		}
	}
}

func (r *Reference) IsResolved() bool {
	return r.Class != nil || r.Variable != nil
}

func (r *Reference) Resolve(state *State) {
	if r.IsResolved() {
		return
	}

	importPath := r.File.FindImportPath(r.Name)

	if importPath == "" {
		for _, v := range r.File.Snapshot().Variables {
			if v.Name == r.Name {
				r.Variable = v
			}
		}

		for _, c := range r.File.Snapshot().Classes {
			if c.Snapshot().Name == r.Name {
				r.Class = c
			}
		}

		return
	}

	var importedFile *File

	extensions := []string{"", ".ts", ".d.ts", ".js"}
	joinSuffixes := []string{"", "index", path.Base(importPath)}

	var err error

	for _, join := range joinSuffixes {
		ip := path.Join(importPath, join)
		for _, extension := range extensions {
			importedFile, err = resolveProjectImportPath(state, r.File, ip+extension)
			if err != nil {
				state.Logger.Println(err)
				break
			}

			if importedFile != nil {
				break
			}
		}

		if importedFile == nil {
			for _, extension := range extensions {
				importedFile, err = resolveNodeModulesImportPath(state, r.File, ip+extension)
				if err != nil {
					state.Logger.Println(err)
					break
				}
				if importedFile != nil {
					break
				}
			}
		}

		if importedFile != nil {
			break
		}
	}

	if importedFile == nil {
		return
	}

	for _, export := range importedFile.Snapshot().Exports {
		if export.Name == r.Name {
			r.Class = export.Class
			r.Variable = export.Variable
		}
	}
}

func resolveIdents(idents []string, file *File, state *State) []*Reference {
	resolved := make([]*Reference, len(idents))

	wg := sync.WaitGroup{}

	for identIndex, ident := range idents {
		wg.Go(func() {
			importPath := file.FindImportPath(ident)

			if importPath == "" {
				return
			}

			var importedFile *File

			extensions := []string{"", ".ts", ".d.ts", ".js"}
			joinSuffixes := []string{"", "index"}

			for _, join := range joinSuffixes {
				ip := path.Join(importPath, join)
				for _, extension := range extensions {
					importedFile, err := resolveProjectImportPath(state, file, ip+extension)
					if err != nil {
						state.Logger.Println(err)
						break
					}

					if importedFile != nil {
						break
					}
				}

				if importedFile == nil {
					for _, extension := range extensions {
						importedFile, err := resolveNodeModulesImportPath(state, file, ip+extension)
						if err != nil {
							state.Logger.Println(err)
							break
						}

						if importedFile != nil {
							break
						}
					}
				}

				if importedFile != nil {
					break
				}
			}

			if importedFile == nil {
				return
			}

			for _, export := range importedFile.Snapshot().Exports {
				if export.Name == ident {
					resolved[identIndex] = export
				}
			}
		})

		if !config.GetConfig().Concurrency {
			wg.Wait()
		}
	}

	wg.Wait()

	return resolved
}

func resolveNodeModulesImportPath(state *State, currentFile *File, importPath string) (*File, error) {
	currentPath := utils.PathDir(FilenameFromUri(currentFile.Snapshot().URI))

	for currentPath != "." && currentPath != "/" {
		nmPath := path.Join(currentPath, "node_modules")
		stat, err := os.Stat(nmPath)
		if err != nil {
			currentPath = utils.PathDir(currentPath)
			continue
		}

		if stat.IsDir() {
			resolvedFile, err := getFileByPath(state, path.Join(nmPath, importPath))
			if err != nil {
				return nil, err
			}

			if resolvedFile != nil {
				return resolvedFile, nil
			}
		}

		currentPath = utils.PathDir(currentPath)
	}

	return nil, nil
}
