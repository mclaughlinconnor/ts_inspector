package pug

import (
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
}

func TestMixinDefinitionUse(t *testing.T) {
	state, err := Parse(`
mixin name(one, two)
  tag
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<ng-template let-one let-two ><tag></tag></ng-template>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
	}
}

func TestMixinDefinitionUseAngular(t *testing.T) {
	state, err := Parse(`
mixin name(one, two)
  tag([attr]='one')
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<ng-template let-one let-two ><tag [attr]='one'  ></tag></ng-template>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
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
}

func TestTagInterpolation(t *testing.T) {
	state, err := Parse(`
tag content #[tag content] content
`)
	got := strings.TrimSuffix(state.HtmlText, "\n")
	want := "<tag><tag>content</tag>content #[tag content] content</tag>"
	if got != want {
		t.Fatalf(`state.HtmlText = '%s', '%v', want '%s'`, got, err, want)
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
}

