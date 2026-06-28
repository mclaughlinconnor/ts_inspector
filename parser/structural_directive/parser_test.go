package structuraldirective_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"ts_inspector/parser/structural_directive"
	"ts_inspector/utils"
)

func makeAs(export string, local string) *structuraldirective.Statement {
	return &structuraldirective.Statement{As: &structuraldirective.As{Local: local, Export: export}}
}

func makeExpression(expression string) *structuraldirective.Statement {
	return &structuraldirective.Statement{Expression: &structuraldirective.Expression{Expression: expression}}
}

func makeKeyExpr(key string, expression string, local string) *structuraldirective.Statement {
	var l *string
	if local == "" {
		l = nil
	} else {
		l = &local
	}

	return &structuraldirective.Statement{KeyExp: &structuraldirective.KeyExp{Key: key, Expression: expression, Local: l}}
}

func makeLet(local string, export string) *structuraldirective.Statement {
	var e *string
	if export == "" {
		e = nil
	} else {
		e = &export
	}

	return &structuraldirective.Statement{Let: &structuraldirective.Let{Local: local, Export: e}}
}

func makeStatements(statements ...*structuraldirective.Statement) *utils.HelpfulArray[*structuraldirective.Statement] {
	return &utils.HelpfulArray[*structuraldirective.Statement]{Elements: statements}
}

func makeShorthand(prefix string, statements *utils.HelpfulArray[*structuraldirective.Statement]) *structuraldirective.ShorthandValue {
	return &structuraldirective.ShorthandValue{Prefix: prefix, Statements: *statements}
}

func TestParseShorthand(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		text    string
		want    *structuraldirective.ShorthandValue
		wantErr bool
	}{
		{
			name: "simple let no trailing",
			text: "let odd",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""))),
		},
		{
			name: "simple let trailing semicolon",
			text: "let odd;",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""))),
		},
		{
			name: "simple let with export",
			text: "let odd=even",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", "even"))),
		},
		{
			name: "chained let with semi colon",
			text: "let odd; let even",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""), makeLet("even", ""))),
		},
		{
			name: "chained let with no delimiter",
			text: "let odd let even",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""), makeLet("even", ""))),
		},
		{
			name: "chained let with comma delimiter",
			text: "let odd, let even;",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""), makeLet("even", ""))),
		},
		{
			name:    "chained let with bad trailing comma delimiter",
			text:    "let odd, let even,",
			wantErr: true,
		},
		{
			name:    "expression followed by lets with bad comma delimiter",
			text:    "true let odd, let even",
			wantErr: true,
		},
		{
			name: "expression followed by lets",
			text: "true let odd; let even",
			want: makeShorthand("prefix", makeStatements(makeExpression("true"), makeLet("odd", ""), makeLet("even", ""))),
		},
		{
			name: "basic as",
			text: "true, a as b",
			want: makeShorthand("prefix", makeStatements(makeExpression("true"), makeAs("a", "b"))),
		},
		{
			name: "as in the middle",
			text: "true, a as b; let c = d",
			want: makeShorthand("prefix", makeStatements(makeExpression("true"), makeAs("a", "b"), makeLet("c", "d"))),
		},
		{
			name: "as with lots of spaces",
			text: "true,    a    as      b; let c = d",
			want: makeShorthand("prefix", makeStatements(makeExpression("true"), makeAs("a", "b"), makeLet("c", "d"))),
		},
		{
			name: "key expr",
			text: "let odd, trackBy: trackByFn",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""), makeKeyExpr("trackBy", "trackByFn", ""))),
		},
		{
			name: "key expr with as local",
			text: "let odd, trackBy: trackByFn as fn",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""), makeKeyExpr("trackBy", "trackByFn", "fn"))),
		},
		{
			name: "key expr with as",
			text: "let odd, trackBy: trackByFn as fn; a as b",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", ""), makeKeyExpr("trackBy", "trackByFn", "fn"), makeAs("a", "b"))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := structuraldirective.ParseShorthand("prefix", tt.text)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ParseShorthand() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("ParseShorthand() succeeded unexpectedly")
			}

			if !reflect.DeepEqual(got, tt.want) {
				gotJson, _ := json.Marshal(got)
				ttWantJson, _ := json.Marshal(tt.want)
				t.Errorf("ParseShorthand() = %+v, want %+v", string(gotJson), string(ttWantJson))
			}
		})
	}
}
