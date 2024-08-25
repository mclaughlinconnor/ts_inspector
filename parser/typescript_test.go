package parser

import (
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"ts_inspector/utils"
)

func extractDeclarationNames(definitions Definitions) []string {
	keys := make([]string, len(definitions))

	i := 0
	for k := range definitions {
		keys[i] = k
		i++
	}

	slices.Sort(keys)

	return keys
}

func TestSimple(t *testing.T) {
	file, err := NewFile("filename", "typescript", 0)
	if err != nil {
		t.Fatal(err)
	}

	file, err = ExtractPugUsages(file, []byte("class Class {}"))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

	if len(file.Usages) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Usages)
	}

	if file.Template != "" {
		t.Fatalf(`Expected origin = "%s" to equal target ""`, file.Template)
	}
}

func TestExtractTemplateFilename(t *testing.T) {
	file, err := NewFile("filename", "typescript", 0)
	if err != nil {
		t.Fatal(err)
	}

	templateUrl := "./template.pug"
	content := fmt.Sprintf(`@Component({templateUrl: '%s'})class Class {}`, templateUrl)
	root, err := utils.GetRootNode(false, content, utils.TypeScript)

	if err != nil {
		t.Fatal(err)
	}

	originalFileExists := utils.FileExists
	utils.FileExists = func(filename string) bool {
		return true
	}
	defer func() { utils.FileExists = originalFileExists }()

	file, err = ExtractTemplateFilename(file, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

	if len(file.Usages) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Usages)
	}

	expectedTemplateUrl, err := filepath.Abs(path.Join(filepath.Dir(file.Filename()), templateUrl))
	if err != nil {
		t.Fatal(err)
	}

	if file.Template != expectedTemplateUrl {
		t.Fatalf(`Expected origin = "%s" to equal target %s`, file.Template, expectedTemplateUrl)
	}
}

func TestExtractTypeScriptUsages(t *testing.T) {
	file, err := NewFile("filename", "typescript", 0)
	if err != nil {
		t.Fatal(err)
	}

	content := `class Class {constructor() {Class.prototype.one; Class.prototype["two"]; this.three; this["four"]; five; Class.six}}`
	root, err := utils.GetRootNode(false, content, utils.TypeScript)

	if err != nil {
		t.Fatal(err)
	}

	originalFileExists := utils.FileExists
	utils.FileExists = func(filename string) bool {
		return true
	}
	defer func() { utils.FileExists = originalFileExists }()

	file, err = ExtractTypeScriptUsages(file, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

	expectedUsages := []string{"four", "one", "three", "two"}
	if !reflect.DeepEqual(extractUsageNames(file.Usages), expectedUsages) {
		t.Fatalf(`Expected origin = %+v to equal target %+v`, extractUsageNames(file.Usages), expectedUsages)
	}

	if file.Template != "" {
		t.Fatalf(`Expected origin = "%s" to equal target ""`, file.Template)
	}
}

func TestExtractTypeScriptDefinitions(t *testing.T) {
	file, err := NewFile("filename", "typescript", 0)
	if err != nil {
		t.Fatal(err)
	}

	content := `class Class {public one = 8; public constructor(private readonly two) {} public get value() {return this.one}}`
	root, err := utils.GetRootNode(false, content, utils.TypeScript)

	if err != nil {
		t.Fatal(err)
	}

	originalFileExists := utils.FileExists
	utils.FileExists = func(filename string) bool {
		return true
	}
	defer func() { utils.FileExists = originalFileExists }()

	file, err = ExtractTypeScriptDefinitions(file, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	expectedDeclarations := []string{"constructor", "one", "two", "value"}
	if !reflect.DeepEqual(extractDeclarationNames(file.Definitions), expectedDeclarations) {
		t.Fatalf(`Expected origin = %+v to equal target %+v`, extractDeclarationNames(file.Definitions), expectedDeclarations)
	}

	if len(file.Usages) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Usages)
	}

	if file.Template != "" {
		t.Fatalf(`Expected origin = "%s" to equal target ""`, file.Template)
	}
}
