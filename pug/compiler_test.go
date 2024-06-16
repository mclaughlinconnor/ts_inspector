package pug

import (
	"reflect"
	"strings"
	"testing"
)

func TestBasicTag(t *testing.T) {
	state, err := Parse("span\n")
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<span></span>"
	if got != want {
		t.Fatalf(`state.HtmlText = %s, %v, want %s`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 0, PugStart: 0, NodeType: EMPTY},
		{HtmlEnd: 5, HtmlStart: 1, PugEnd: 4, PugStart: 0, NodeType: TAG_NAME},
		{HtmlEnd: 6, HtmlStart: 6, PugEnd: 4, PugStart: 4, NodeType: EMPTY},
		{HtmlEnd: 14, HtmlStart: 13, PugEnd: 5, PugStart: 5, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(2, state)
	var target uint32 = 1
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestSeveralBasicTag(t *testing.T) {
	state, err := Parse(`
div
div
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<div></div><div></div>"
	if got != want {
		t.Fatalf(`state.HtmlText = %s, %v, want %s`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 1, PugStart: 1, NodeType: EMPTY},
		{HtmlEnd: 4, HtmlStart: 1, PugEnd: 4, PugStart: 1, NodeType: TAG_NAME},
		{HtmlEnd: 5, HtmlStart: 5, PugEnd: 4, PugStart: 4, NodeType: EMPTY},
		{HtmlEnd: 11, HtmlStart: 11, PugEnd: 5, PugStart: 5, NodeType: EMPTY},
		{HtmlEnd: 15, HtmlStart: 12, PugEnd: 8, PugStart: 5, NodeType: TAG_NAME},
		{HtmlEnd: 16, HtmlStart: 16, PugEnd: 8, PugStart: 8, NodeType: EMPTY},
		{HtmlEnd: 23, HtmlStart: 22, PugEnd: 9, PugStart: 9, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(13, state)
	var target uint32 = 6
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestBasicTagWithAttribute(t *testing.T) {
	state, err := Parse(`
div(attr=true)
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<div attr='true'  ></div>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 1, PugStart: 1, NodeType: EMPTY},
		{HtmlEnd: 4, HtmlStart: 1, PugEnd: 4, PugStart: 1, NodeType: TAG_NAME},
		{HtmlEnd: 5, HtmlStart: 4, PugEnd: 4, PugStart: 4, NodeType: SPACE},
		{HtmlEnd: 9, HtmlStart: 5, PugEnd: 9, PugStart: 5, NodeType: ATTRIBUTE_NAME},
		{HtmlEnd: 10, HtmlStart: 9, PugEnd: 10, PugStart: 10, NodeType: EQUALS},
		{HtmlEnd: 15, HtmlStart: 11, PugEnd: 14, PugStart: 10, NodeType: JAVASCRIPT},
		{HtmlEnd: 17, HtmlStart: 16, PugEnd: 14, PugStart: 15, NodeType: SPACE},
		{HtmlEnd: 18, HtmlStart: 17, PugEnd: 15, PugStart: 14, NodeType: SPACE},
		{HtmlEnd: 19, HtmlStart: 19, PugEnd: 15, PugStart: 15, NodeType: EMPTY},
		{HtmlEnd: 26, HtmlStart: 25, PugEnd: 16, PugStart: 16, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(12, state)
	var target uint32 = 11
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestBasicTagWithAttributes(t *testing.T) {
	state, err := Parse(`
div(attr=true, attr=false)
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<div attr='true'  attr='false'  ></div>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 1, PugStart: 1, NodeType: EMPTY},
		{HtmlEnd: 4, HtmlStart: 1, PugEnd: 4, PugStart: 1, NodeType: TAG_NAME},
		{HtmlEnd: 5, HtmlStart: 4, PugEnd: 4, PugStart: 4, NodeType: SPACE},
		{HtmlEnd: 9, HtmlStart: 5, PugEnd: 9, PugStart: 5, NodeType: ATTRIBUTE_NAME},
		{HtmlEnd: 10, HtmlStart: 9, PugEnd: 10, PugStart: 10, NodeType: EQUALS},
		{HtmlEnd: 15, HtmlStart: 11, PugEnd: 14, PugStart: 10, NodeType: JAVASCRIPT},
		{HtmlEnd: 17, HtmlStart: 16, PugEnd: 14, PugStart: 16, NodeType: SPACE},
		{HtmlEnd: 18, HtmlStart: 17, PugEnd: 15, PugStart: 15, NodeType: SPACE},
		{HtmlEnd: 22, HtmlStart: 18, PugEnd: 20, PugStart: 16, NodeType: ATTRIBUTE_NAME},
		{HtmlEnd: 23, HtmlStart: 22, PugEnd: 21, PugStart: 21, NodeType: EQUALS},
		{HtmlEnd: 29, HtmlStart: 24, PugEnd: 26, PugStart: 21, NodeType: JAVASCRIPT},
		{HtmlEnd: 31, HtmlStart: 30, PugEnd: 26, PugStart: 27, NodeType: SPACE},
		{HtmlEnd: 32, HtmlStart: 31, PugEnd: 27, PugStart: 26, NodeType: SPACE},
		{HtmlEnd: 33, HtmlStart: 33, PugEnd: 27, PugStart: 27, NodeType: EMPTY},
		{HtmlEnd: 40, HtmlStart: 39, PugEnd: 28, PugStart: 28, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(25, state)
	var target uint32 = 22
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestBasicTagWithComplexAttributes(t *testing.T) {
	state, err := Parse(`
div(attr='"true"', attr=false+'24')
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<div attr='\"true\"'  attr=\"false+'24'\"  ></div>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 1, PugStart: 1, NodeType: EMPTY},
		{HtmlEnd: 4, HtmlStart: 1, PugEnd: 4, PugStart: 1, NodeType: TAG_NAME},
		{HtmlEnd: 5, HtmlStart: 4, PugEnd: 4, PugStart: 4, NodeType: SPACE},
		{HtmlEnd: 9, HtmlStart: 5, PugEnd: 9, PugStart: 5, NodeType: ATTRIBUTE_NAME},
		{HtmlEnd: 10, HtmlStart: 9, PugEnd: 10, PugStart: 10, NodeType: EQUALS},
		{HtmlEnd: 18, HtmlStart: 10, PugEnd: 18, PugStart: 10, NodeType: ATTRIBUTE},
		{HtmlEnd: 19, HtmlStart: 18, PugEnd: 18, PugStart: 20, NodeType: SPACE},
		{HtmlEnd: 20, HtmlStart: 19, PugEnd: 19, PugStart: 19, NodeType: SPACE},
		{HtmlEnd: 24, HtmlStart: 20, PugEnd: 24, PugStart: 20, NodeType: ATTRIBUTE_NAME},
		{HtmlEnd: 25, HtmlStart: 24, PugEnd: 25, PugStart: 25, NodeType: EQUALS},
		{HtmlEnd: 36, HtmlStart: 26, PugEnd: 35, PugStart: 25, NodeType: JAVASCRIPT},
		{HtmlEnd: 38, HtmlStart: 37, PugEnd: 35, PugStart: 36, NodeType: SPACE},
		{HtmlEnd: 39, HtmlStart: 38, PugEnd: 36, PugStart: 35, NodeType: SPACE},
		{HtmlEnd: 40, HtmlStart: 40, PugEnd: 36, PugStart: 36, NodeType: EMPTY},
		{HtmlEnd: 47, HtmlStart: 46, PugEnd: 37, PugStart: 37, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(31, state)
	var target uint32 = 30
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestMixinDefinition(t *testing.T) {
	state, err := Parse(`
mixin name(one, two)
  tag
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<ng-template let-one let-two ><tag></tag></ng-template>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 20, HtmlStart: 17, PugEnd: 15, PugStart: 12, NodeType: ATTRIBUTE},
		{HtmlEnd: 28, HtmlStart: 25, PugEnd: 20, PugStart: 17, NodeType: ATTRIBUTE},
		{HtmlEnd: 30, HtmlStart: 30, PugEnd: 24, PugStart: 24, NodeType: EMPTY},
		{HtmlEnd: 34, HtmlStart: 31, PugEnd: 27, PugStart: 24, NodeType: TAG_NAME},
		{HtmlEnd: 35, HtmlStart: 35, PugEnd: 27, PugStart: 27, NodeType: EMPTY},
		{HtmlEnd: 56, HtmlStart: 55, PugEnd: 28, PugStart: 28, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(27, state)
	var target uint32 = 19
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestMixinDefinitionUse(t *testing.T) {
	state, err := Parse(`
mixin name(one, two)
  tag&attributes(attributes)
+name('one', 2)([attr]='value')
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<ng-template let-one let-two ><tag  ></tag></ng-template>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 20, HtmlStart: 17, PugEnd: 15, PugStart: 12, NodeType: ATTRIBUTE},
		{HtmlEnd: 28, HtmlStart: 25, PugEnd: 20, PugStart: 17, NodeType: ATTRIBUTE},
		{HtmlEnd: 30, HtmlStart: 30, PugEnd: 24, PugStart: 24, NodeType: EMPTY},
		{HtmlEnd: 34, HtmlStart: 31, PugEnd: 27, PugStart: 24, NodeType: TAG_NAME},
		{HtmlEnd: 35, HtmlStart: 34, PugEnd: 27, PugStart: 27, NodeType: SPACE},
		{HtmlEnd: 36, HtmlStart: 35, PugEnd: 50, PugStart: 49, NodeType: SPACE},
		{HtmlEnd: 37, HtmlStart: 37, PugEnd: 50, PugStart: 50, NodeType: EMPTY},
		{HtmlEnd: 58, HtmlStart: 57, PugEnd: 83, PugStart: 51, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(27, state)
	var target uint32 = 19
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestMixinDefinitionUseAngular(t *testing.T) {
	state, err := Parse(`
mixin name(one, two)
  tag([attr]='one')
+name('one', 'two')
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<ng-template let-one let-two ><tag [attr]='one'  ></tag></ng-template>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 20, HtmlStart: 17, PugEnd: 15, PugStart: 12, NodeType: ATTRIBUTE},
		{HtmlEnd: 28, HtmlStart: 25, PugEnd: 20, PugStart: 17, NodeType: ATTRIBUTE},
		{HtmlEnd: 30, HtmlStart: 30, PugEnd: 24, PugStart: 24, NodeType: EMPTY},
		{HtmlEnd: 34, HtmlStart: 31, PugEnd: 27, PugStart: 24, NodeType: TAG_NAME},
		{HtmlEnd: 35, HtmlStart: 34, PugEnd: 27, PugStart: 27, NodeType: SPACE},
		{HtmlEnd: 41, HtmlStart: 35, PugEnd: 34, PugStart: 28, NodeType: ANGULAR_ATTRIBUTE_NAME},
		{HtmlEnd: 42, HtmlStart: 41, PugEnd: 35, PugStart: 35, NodeType: EQUALS},
		{HtmlEnd: 47, HtmlStart: 42, PugEnd: 40, PugStart: 35, NodeType: ATTRIBUTE},
		{HtmlEnd: 48, HtmlStart: 47, PugEnd: 40, PugStart: 41, NodeType: SPACE},
		{HtmlEnd: 49, HtmlStart: 48, PugEnd: 41, PugStart: 40, NodeType: SPACE},
		{HtmlEnd: 50, HtmlStart: 50, PugEnd: 41, PugStart: 41, NodeType: EMPTY},
		{HtmlEnd: 71, HtmlStart: 70, PugEnd: 62, PugStart: 42, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(27, state)
	var target uint32 = 19
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestLoops(t *testing.T) {
	state, err := Parse(`
each x in [1, 2, 3]
  tag x
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<script>return x ;</script><script>return [1, 2, 3];</script><tag>x</tag>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 17, HtmlStart: 15, PugEnd: 8, PugStart: 6, NodeType: JAVASCRIPT},
		{HtmlEnd: 51, HtmlStart: 42, PugEnd: 20, PugStart: 11, NodeType: JAVASCRIPT},
		{HtmlEnd: 61, HtmlStart: 61, PugEnd: 23, PugStart: 23, NodeType: EMPTY},
		{HtmlEnd: 65, HtmlStart: 62, PugEnd: 26, PugStart: 23, NodeType: TAG_NAME},
		{HtmlEnd: 66, HtmlStart: 66, PugEnd: 26, PugStart: 26, NodeType: EMPTY},
		{HtmlEnd: 67, HtmlStart: 66, PugEnd: 28, PugStart: 27, NodeType: CONTENT},
		{HtmlEnd: 74, HtmlStart: 73, PugEnd: 29, PugStart: 29, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(46, state)
	var target uint32 = 15
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestScripts(t *testing.T) {
	state, err := Parse(`
script.
  const x = 124; console.log(x);
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<script>  const x = 124; console.log(x);\n;</script>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 41, HtmlStart: 8, PugEnd: 42, PugStart: 9, NodeType: JAVASCRIPT},
		{HtmlEnd: 52, HtmlStart: 51, PugEnd: 42, PugStart: 43, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(16, state)
	var target uint32 = 17
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestUnescapedBufferedCode(t *testing.T) {
	state, err := Parse(`
tag!= 2+2
tag= 2+2
- const x = 2+2
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<tag><script>return  2+2;</script></tag><tag><script>return  2+2;</script></tag><script>const x = 2+2;</script>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 1, PugStart: 1, NodeType: EMPTY},
		{HtmlEnd: 4, HtmlStart: 1, PugEnd: 4, PugStart: 1, NodeType: TAG_NAME},
		{HtmlEnd: 5, HtmlStart: 5, PugEnd: 4, PugStart: 4, NodeType: EMPTY},
		{HtmlEnd: 24, HtmlStart: 20, PugEnd: 10, PugStart: 6, NodeType: JAVASCRIPT},
		{HtmlEnd: 40, HtmlStart: 40, PugEnd: 11, PugStart: 11, NodeType: EMPTY},
		{HtmlEnd: 44, HtmlStart: 41, PugEnd: 14, PugStart: 11, NodeType: TAG_NAME},
		{HtmlEnd: 45, HtmlStart: 45, PugEnd: 14, PugStart: 14, NodeType: EMPTY},
		{HtmlEnd: 64, HtmlStart: 60, PugEnd: 19, PugStart: 15, NodeType: JAVASCRIPT},
		{HtmlEnd: 101, HtmlStart: 88, PugEnd: 35, PugStart: 22, NodeType: JAVASCRIPT},
		{HtmlEnd: 112, HtmlStart: 111, PugEnd: 36, PugStart: 36, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(63, state)
	var target uint32 = 18
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestPipe(t *testing.T) {
	state, err := Parse(`
tag content
  | text two
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<tag>content text two</tag>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 1, PugStart: 1, NodeType: EMPTY},
		{HtmlEnd: 4, HtmlStart: 1, PugEnd: 4, PugStart: 1, NodeType: TAG_NAME},
		{HtmlEnd: 5, HtmlStart: 5, PugEnd: 4, PugStart: 4, NodeType: EMPTY},
		{HtmlEnd: 12, HtmlStart: 5, PugEnd: 12, PugStart: 5, NodeType: CONTENT},
		{HtmlEnd: 21, HtmlStart: 12, PugEnd: 25, PugStart: 16, NodeType: CONTENT},
		{HtmlEnd: 28, HtmlStart: 27, PugEnd: 26, PugStart: 26, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(19, state)
	var target uint32 = 23
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestTagInterpolation(t *testing.T) {
	state, err := Parse(`
tag one #[tag two] three
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<tag><tag>two</tag>one #[tag two] three</tag>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 0, HtmlStart: 0, PugEnd: 1, PugStart: 1, NodeType: EMPTY},
		{HtmlEnd: 4, HtmlStart: 1, PugEnd: 4, PugStart: 1, NodeType: TAG_NAME},
		{HtmlEnd: 5, HtmlStart: 5, PugEnd: 4, PugStart: 4, NodeType: EMPTY},
		{HtmlEnd: 5, HtmlStart: 5, PugEnd: 11, PugStart: 11, NodeType: EMPTY},
		{HtmlEnd: 9, HtmlStart: 6, PugEnd: 14, PugStart: 11, NodeType: TAG_NAME},
		{HtmlEnd: 10, HtmlStart: 10, PugEnd: 14, PugStart: 14, NodeType: EMPTY},
		{HtmlEnd: 13, HtmlStart: 10, PugEnd: 18, PugStart: 15, NodeType: CONTENT},
		{HtmlEnd: 39, HtmlStart: 19, PugEnd: 25, PugStart: 5, NodeType: CONTENT},
		{HtmlEnd: 46, HtmlStart: 45, PugEnd: 26, PugStart: 26, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(11, state)
	var target uint32 = 16
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestExtendsIncludes(t *testing.T) {
	state, err := Parse(`
extends layout
append head
  tag
  tag
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<a href=' layout'></a><tag></tag><tag></tag>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 16, HtmlStart: 9, PugEnd: 15, PugStart: 8, NodeType: FILENAME},
		{HtmlEnd: 22, HtmlStart: 22, PugEnd: 30, PugStart: 30, NodeType: EMPTY},
		{HtmlEnd: 26, HtmlStart: 23, PugEnd: 33, PugStart: 30, NodeType: TAG_NAME},
		{HtmlEnd: 27, HtmlStart: 27, PugEnd: 33, PugStart: 33, NodeType: EMPTY},
		{HtmlEnd: 33, HtmlStart: 33, PugEnd: 36, PugStart: 36, NodeType: EMPTY},
		{HtmlEnd: 37, HtmlStart: 34, PugEnd: 39, PugStart: 36, NodeType: TAG_NAME},
		{HtmlEnd: 38, HtmlStart: 38, PugEnd: 39, PugStart: 39, NodeType: EMPTY},
		{HtmlEnd: 45, HtmlStart: 44, PugEnd: 40, PugStart: 40, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(11, state)
	var target uint32 = 10
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestCase(t *testing.T) {
	state, err := Parse(`
case data
  when 0
  when 1
    tag
  default
    tag
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<script>return data;</script><script>return 0;</script><script>return 1;</script><tag></tag><tag></tag>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}

	expectedRanges := []Range{
		{HtmlEnd: 19, HtmlStart: 15, PugEnd: 10, PugStart: 6, NodeType: JAVASCRIPT},
		{HtmlEnd: 45, HtmlStart: 44, PugEnd: 19, PugStart: 18, NodeType: JAVASCRIPT},
		{HtmlEnd: 71, HtmlStart: 70, PugEnd: 28, PugStart: 27, NodeType: JAVASCRIPT},
		{HtmlEnd: 81, HtmlStart: 81, PugEnd: 33, PugStart: 33, NodeType: EMPTY},
		{HtmlEnd: 85, HtmlStart: 82, PugEnd: 36, PugStart: 33, NodeType: TAG_NAME},
		{HtmlEnd: 86, HtmlStart: 86, PugEnd: 36, PugStart: 36, NodeType: EMPTY},
		{HtmlEnd: 92, HtmlStart: 92, PugEnd: 51, PugStart: 51, NodeType: EMPTY},
		{HtmlEnd: 96, HtmlStart: 93, PugEnd: 54, PugStart: 51, NodeType: TAG_NAME},
		{HtmlEnd: 97, HtmlStart: 97, PugEnd: 54, PugStart: 54, NodeType: EMPTY},
		{HtmlEnd: 104, HtmlStart: 103, PugEnd: 55, PugStart: 55, NodeType: EMPTY},
	}

	if !reflect.DeepEqual(state.Ranges, expectedRanges) {
		t.Fatalf(`state.Ranges = %+v, want %+v`, state.Ranges, expectedRanges)
	}

	origin := HtmlLocationToPugLocation(44, state)
	var target uint32 = 18
	if origin != target {
		t.Fatalf(`Expected origin = %d to equal target %d`, origin, target)
	}
}

func TestJavascriptTemplateStringAttributes(t *testing.T) {
	state, err := Parse(`
input.form-control([placeholder]=` + "`hello`" + `)
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := `<input class='form-control' [placeholder]="$any('hello')"  />`
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}
}

func TestJavascriptVariableAttributes(t *testing.T) {
	state, err := Parse(`
tag([input]=variable, attr=variable)
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := `<tag [input]="$any('variable')"  attr='variable'  ></tag>`
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}
}
