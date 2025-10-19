package parser

import (
	"os"
	"reflect"
	"slices"
	"testing"
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

	// Create parent class
	parent := &Class{Name: "Parent"}
	state.Classes["parent-id"] = parent

	// Create interface classes
	interface1 := &Class{Name: "Interface1"}
	state.Classes["interface1-id"] = interface1

	interface2 := &Class{Name: "Interface2"}
	state.Classes["interface2-id"] = interface2

	// Create child class with extends and implements
	child := &Class{
		Name:                  "Child",
		ExtendsClauseName:     "Parent",
		ImplementsClauseNames: []string{"Interface1", "Interface2"},
	}
	state.Classes["child-id"] = child

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
