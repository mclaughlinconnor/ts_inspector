package parser

import (
	"os"
	"reflect"
	"slices"
	"testing"
	"ts_inspector/ast"
	"ts_inspector/utils"
)

func TestMain(m *testing.M) {
	utils.InitQueries()

	code := m.Run()
	os.Exit(code)
}

func TestExtractExtendsImplements(t *testing.T) {
	content := `class Child extends Parent implements Interface1, Interface2 {}`
	root, err := utils.GetRootNode(false, content, utils.TypeScript)
	if err != nil {
		t.Fatal(err)
	}

	class := Class{Content: content}
	class, err = ExtractClassName(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	class, err = ExtractExtendsImplements(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	if class.ExtendsClauseName != "Parent" {
		t.Fatalf(`Expected ExtendsClauseName = "%s" to equal "Parent"`, class.ExtendsClauseName)
	}

	expectedImplements := []string{"Interface1", "Interface2"}
	if !reflect.DeepEqual(class.ImplementsClauseNames, expectedImplements) {
		t.Fatalf(`Expected ImplementsClauseNames = %+v to equal %+v`, class.ImplementsClauseNames, expectedImplements)
	}
}

func TestExtractExtendsOnly(t *testing.T) {
	content := `class Child extends Parent {}`
	root, err := utils.GetRootNode(false, content, utils.TypeScript)
	if err != nil {
		t.Fatal(err)
	}

	class := Class{Content: content}
	class, err = ExtractClassName(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	class, err = ExtractExtendsImplements(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	if class.ExtendsClauseName != "Parent" {
		t.Fatalf(`Expected ExtendsClauseName = "%s" to equal "Parent"`, class.ExtendsClauseName)
	}

	if len(class.ImplementsClauseNames) != 0 {
		t.Fatalf(`Expected ImplementsClauseNames = %+v to be empty`, class.ImplementsClauseNames)
	}
}

func TestExtractImplementsOnly(t *testing.T) {
	content := `class MyClass implements Interface1 {}`
	root, err := utils.GetRootNode(false, content, utils.TypeScript)
	if err != nil {
		t.Fatal(err)
	}

	class := Class{Content: content}
	class, err = ExtractClassName(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	class, err = ExtractExtendsImplements(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	if class.ExtendsClauseName != "" {
		t.Fatalf(`Expected ExtendsClauseName = "%s" to be empty`, class.ExtendsClauseName)
	}

	expectedImplements := []string{"Interface1"}
	if !reflect.DeepEqual(class.ImplementsClauseNames, expectedImplements) {
		t.Fatalf(`Expected ImplementsClauseNames = %+v to equal %+v`, class.ImplementsClauseNames, expectedImplements)
	}
}

func TestExtractNoExtendsOrImplements(t *testing.T) {
	content := `class MyClass {}`
	root, err := utils.GetRootNode(false, content, utils.TypeScript)
	if err != nil {
		t.Fatal(err)
	}

	class := Class{Content: content}
	class, err = ExtractClassName(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	class, err = ExtractExtendsImplements(class, root, []byte(content))
	if err != nil {
		t.Fatal(err)
	}

	if class.ExtendsClauseName != "" {
		t.Fatalf(`Expected ExtendsClauseName = "%s" to be empty`, class.ExtendsClauseName)
	}

	if len(class.ImplementsClauseNames) != 0 {
		t.Fatalf(`Expected ImplementsClauseNames = %+v to be empty`, class.ImplementsClauseNames)
	}
}

func TestResolveClassRelationships(t *testing.T) {
	state := State{
		Classes: map[string]*Class{},
		Files:   map[string]*File{},
	}

	// Create a file for the parent and interfaces
	parentFile := &File{
		URI:     "file:///test/parent.ts",
		Classes: []*Class{},
	}
	state.Files["/test/parent.ts"] = parentFile

	// Create parent class
	parent := &Class{Name: "Parent", File: parentFile}
	parentFile.Classes = append(parentFile.Classes, parent)
	state.Classes["file:///test/parent.ts-Parent"] = parent

	// Create interface classes
	interface1 := &Class{Name: "Interface1", File: parentFile}
	parentFile.Classes = append(parentFile.Classes, interface1)
	state.Classes["file:///test/parent.ts-Interface1"] = interface1

	interface2 := &Class{Name: "Interface2", File: parentFile}
	parentFile.Classes = append(parentFile.Classes, interface2)
	state.Classes["file:///test/parent.ts-Interface2"] = interface2

	// Create a file for the child
	childFile := &File{
		URI:     "file:///test/child.ts",
		Classes: []*Class{},
	}
	state.Files["/test/child.ts"] = childFile

	// Create child class with extends and implements in the same file (no imports needed)
	child := &Class{
		Name:                  "Child",
		File:                  childFile,
		ExtendsClauseName:     "Parent",
		ImplementsClauseNames: []string{"Interface1", "Interface2"},
	}
	childFile.Classes = append(childFile.Classes, child)
	state.Classes["file:///test/child.ts-Child"] = child

	// Also add Parent to child file so it resolves (simulating same-file inheritance)
	parentInChildFile := &Class{Name: "Parent", File: childFile}
	childFile.Classes = append(childFile.Classes, parentInChildFile)

	interface1InChildFile := &Class{Name: "Interface1", File: childFile}
	childFile.Classes = append(childFile.Classes, interface1InChildFile)

	interface2InChildFile := &Class{Name: "Interface2", File: childFile}
	childFile.Classes = append(childFile.Classes, interface2InChildFile)

	// Resolve relationships
	state.ResolveClassRelationships()

	// Check extends relationship
	if child.Extends == nil {
		t.Fatal("Expected child.Extends to be set")
	}
	if child.Extends.Name != "Parent" {
		t.Fatalf(`Expected child.Extends.Name = "%s" to equal "Parent"`, child.Extends.Name)
	}

	// Check implements relationships
	if len(child.Implements) != 2 {
		t.Fatalf(`Expected len(child.Implements) = %d to equal 2`, len(child.Implements))
	}

	implementNames := []string{child.Implements[0].Name, child.Implements[1].Name}
	slices.Sort(implementNames)
	expectedNames := []string{"Interface1", "Interface2"}
	if !reflect.DeepEqual(implementNames, expectedNames) {
		t.Fatalf(`Expected implements names = %+v to equal %+v`, implementNames, expectedNames)
	}
}

func TestResolveClassRelationshipsWithImports(t *testing.T) {
	state := State{
		Classes: map[string]*Class{},
		Files:   map[string]*File{},
	}

	// Create a file for the parent
	parentFile := &File{
		URI:     "file:///test/parent.ts",
		Classes: []*Class{},
	}
	state.Files["/test/parent.ts"] = parentFile

	// Create parent class
	parent := &Class{Name: "BaseClass", File: parentFile}
	parentFile.Classes = append(parentFile.Classes, parent)
	state.Classes["file:///test/parent.ts-BaseClass"] = parent

	// Create interface class
	iface := &Class{Name: "IInterface", File: parentFile}
	parentFile.Classes = append(parentFile.Classes, iface)
	state.Classes["file:///test/parent.ts-IInterface"] = iface

	// Create a file for the child with imports
	childFile := &File{
		URI:     "file:///test/child.ts",
		Classes: []*Class{},
		Imports: []*ast.ImportParseResult{
			{
				Package: "./parent",
				Imports: []ast.ImportIdentifier{
					{
						ForeignIdentifier: "BaseClass",
						LocalIdentifier:   "BaseClass",
						IsType:            false,
					},
					{
						ForeignIdentifier: "IInterface",
						LocalIdentifier:   "IInterface",
						IsType:            false,
					},
				},
			},
		},
	}
	state.Files["/test/child.ts"] = childFile

	// Create child class with extends and implements from imports
	child := &Class{
		Name:                  "ChildClass",
		File:                  childFile,
		ExtendsClauseName:     "BaseClass",
		ImplementsClauseNames: []string{"IInterface"},
	}
	childFile.Classes = append(childFile.Classes, child)
	state.Classes["file:///test/child.ts-ChildClass"] = child

	// Mock FileExists to return true for our test files
	originalFileExists := utils.FileExists
	utils.FileExists = func(filename string) bool {
		return filename == "/test/parent.ts" || filename == "/test/child.ts"
	}
	defer func() { utils.FileExists = originalFileExists }()

	// Resolve relationships
	state.ResolveClassRelationships()

	// Check extends relationship
	if child.Extends == nil {
		t.Fatal("Expected child.Extends to be set")
	}
	if child.Extends.Name != "BaseClass" {
		t.Fatalf(`Expected child.Extends.Name = "%s" to equal "BaseClass"`, child.Extends.Name)
	}
	if child.Extends.File.URI != "file:///test/parent.ts" {
		t.Fatalf(`Expected child.Extends to be from parent.ts, got %s`, child.Extends.File.URI)
	}

	// Check implements relationships
	if len(child.Implements) != 1 {
		t.Fatalf(`Expected len(child.Implements) = %d to equal 1`, len(child.Implements))
	}
	if child.Implements[0].Name != "IInterface" {
		t.Fatalf(`Expected child.Implements[0].Name = "%s" to equal "IInterface"`, child.Implements[0].Name)
	}
	if child.Implements[0].File.URI != "file:///test/parent.ts" {
		t.Fatalf(`Expected child.Implements[0] to be from parent.ts, got %s`, child.Implements[0].File.URI)
	}
}
