package structuraldirective_test

import (
	"testing"
	structuraldirective "ts_inspector/parser/structural_directive"
)

func Test_parseExpression(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		text    string
		want    string
		wantErr bool
	}{
		{name: "1", text: "kind + true + {a: b, b: c} + let (false).y let x = id", want: "kind + true + {a: b, b: c} + let (false).y"},
		{name: "2", text: "a + b", want: "a + b"},
		{name: "3", text: "a", want: "a"},
		{name: "4", text: "call(a, b, c)", want: "call(a, b, c)"},
		{name: "5", text: "[one, two, three]", want: "[one, two, three]"},
		{name: "6", text: "[one + two, two+three, three +four, four+ five]", want: "[one + two, two+three, three +four, four+ five]"},
		{name: "7", text: "true, trackBy: [one + two, two+three, three +four, four+ five]", want: "true"},
		{name: "8", text: "[one + two, two+three, three +four, four+ five] trackBy: func", want: "[one + two, two+three, three +four, four+ five]"},
		{name: "9", text: "one + two, two+three, three +four, four+ five trackBy: func", want: "one + two"},
		{name: "10", text: "one + two - two+three - three +four - four+ five trackBy: func", want: "one + two - two+three - three +four - four+ five"},
		{name: "11", text: "four+ five trackBy: func", want: "four+ five"},
		{name: "12", text: "5", want: "5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := structuraldirective.ParseExpression(0, []rune(tt.text))
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("parseExpression() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("parseExpression() succeeded unexpectedly")
			}

			gotText := string(tt.text[0:got])
			if gotText != tt.want {
				t.Errorf("parseExpression() = %q, want %q", gotText, tt.want)
			}
		})
	}
}
