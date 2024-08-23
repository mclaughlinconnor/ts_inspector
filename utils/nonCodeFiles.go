package utils

import "strings"

func RemoveNonCode(files []string) []string {
	var code = []string{}
	for _, file := range files {
		if !strings.HasSuffix(file, ".d.ts") && !strings.HasSuffix(file, ".spec.ts") {
			code = append(code, file)
		}
	}

	return code
}
