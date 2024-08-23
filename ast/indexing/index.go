package indexing

func Index(rootUri string) []string {
	roots := findProjectRoots(rootUri)

	imports := []string{}
	for _, root := range roots {
		i, e := recursivelyRetrieveImports(root, 0, 1)
		if e == nil {
			imports = append(imports, i...)
		}
	}

	return imports
}
