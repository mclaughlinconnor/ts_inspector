package parser

import (
	"os"
	"reflect"
	"testing"
	"ts_inspector/utils"
)

func extractUsageNames(usages Usages) []string {
	keys := make([]string, len(usages))

	i := 0
	for k := range usages {
		keys[i] = k
		i++
	}

	return keys
}

func TestMain(m *testing.M) {
	utils.InitQueries()

	code := m.Run()
	// teardown here
	os.Exit(code)
}

func TestExtractPugUsagesSimple(t *testing.T) {
	file, err := NewFile("filename", "pug", 0)
	if err != nil {
		t.Fatal(err)
	}

	file, err = ExtractPugUsages(file, []byte("tag"))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

	if len(file.Usages) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Usages)
	}
}

func TestExtractPugUsagesNonAngularAttrs(t *testing.T) {
	file, err := NewFile("filename", "pug", 0)
	if err != nil {
		t.Fatal(err)
	}

	file, err = ExtractPugUsages(file, []byte("tag(attr='value', attr=\"value\")"))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

	if len(file.Usages) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Usages)
	}
}

func TestExtractPugUsagesNonAngularContent(t *testing.T) {
	file, err := NewFile("filename", "pug", 0)
	if err != nil {
		t.Fatal(err)
	}

	file, err = ExtractPugUsages(file, []byte("tag content #{content} content"))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

	if len(file.Usages) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Usages)
	}
}

func TestExtractPugUsagesAngularAttributes(t *testing.T) {
	file, err := NewFile("filename", "pug", 0)
	if err != nil {
		t.Fatal(err)
	}

	file, err = ExtractPugUsages(file, []byte("tag((change)='onChange($event)', [ngClass]='clazz', [(ngModel)]='value')"))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

  target := []string{"onChange", "$event", "clazz", "value"}	
  if !reflect.DeepEqual(extractUsageNames(file.Usages), target) {
		t.Fatalf(`Expected origin = %+v to equal target %+v`, extractUsageNames(file.Usages), target)
	}
}

func TestExtractPugUsagesAngularContent(t *testing.T) {
	file, err := NewFile("filename", "pug", 0)
	if err != nil {
		t.Fatal(err)
	}

	file, err = ExtractPugUsages(file, []byte("tag content {{one}} content {{two|three:four}}"))
	if err != nil {
		t.Fatal(err)
	}

	if len(file.Definitions) != 0 {
		t.Fatalf(`Expected origin = %+v to equal target []`, file.Definitions)
	}

  target := []string{"one", "two", "three", "four"}
	if !reflect.DeepEqual(extractUsageNames(file.Usages), target) {
		t.Fatalf(`Expected origin = %+v to equal target %+v`, extractUsageNames(file.Usages), target)
	}
}
