package structuraldirective_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"ts_inspector/parser/structural_directive"
	"ts_inspector/utils"
)

func makeExpression(expression string, expressionOffset int, local string, localOffset int) *structuraldirective.Statement {
	var l *string
	if local == "" {
		l = nil
	} else {
		l = &local
	}

	return &structuraldirective.Statement{Expression: &structuraldirective.Expression{Expression: expression, ExpressionOffset: expressionOffset, Local: l, LocalOffset: localOffset}}
}

func makeKeyExpr(key string, keyOffset int, expression string, expressionOffset int, local string, localOffset int) *structuraldirective.Statement {
	var l *string
	if local == "" {
		l = nil
	} else {
		l = &local
	}

	return &structuraldirective.Statement{KeyExp: &structuraldirective.KeyExp{Key: key, KeyOffset: keyOffset, Expression: expression, ExpressionOffset: expressionOffset, Local: l, LocalOffset: localOffset}}
}

func makeLet(local string, localOffset int, export string, exportOffset int) *structuraldirective.Statement {
	var e *string
	if export == "" {
		e = nil
	} else {
		e = &export
	}

	return &structuraldirective.Statement{Let: &structuraldirective.Let{Local: local, LocalOffset: localOffset, Export: e, ExportOffset: exportOffset}}
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
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0))),
		},
		{
			name: "simple let trailing semicolon",
			text: "let odd;",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0))),
		},
		{
			name: "simple let with export",
			text: "let odd=even",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "even", 8))),
		},
		{
			name: "chained let with semi colon",
			text: "let odd; let even",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0), makeLet("even", 13, "", 0))),
		},
		{
			name: "chained let with no delimiter",
			text: "let odd let even",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0), makeLet("even", 12, "", 0))),
		},
		{
			name: "chained let with comma delimiter",
			text: "let odd, let even;",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0), makeLet("even", 13, "", 0))),
		},
		{
			name: "chained let with previously bad trailing comma delimiter", // the spec says the comma is wrong, but the compiler parses it fine
			text: "let odd, let even,",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0), makeLet("even", 13, "", 0))),
		},
		{
			name: "expression followed by lets with previously bad comma delimiter", // the spec says the comma is wrong, but the compiler parses it fine
			text: "true let odd, let even",
			want: makeShorthand("prefix", makeStatements(makeExpression("true", 0, "", 0), makeLet("odd", 9, "", 0), makeLet("even", 18, "", 0))),
		},
		{
			name: "expression followed by lets",
			text: "true let odd; let even",
			want: makeShorthand("prefix", makeStatements(makeExpression("true", 0, "", 0), makeLet("odd", 9, "", 0), makeLet("even", 18, "", 0))),
		},
		{
			name: "basic as",
			text: "true, a as b",
			want: makeShorthand("prefix", makeStatements(makeExpression("true", 0, "", 0), makeExpression("a", 6, "b", 11))),
		},
		{
			name: "as in the middle",
			text: "true, a as b; let c = d",
			want: makeShorthand("prefix", makeStatements(makeExpression("true", 0, "", 0), makeExpression("a", 6, "b", 11), makeLet("c", 18, "d", 22))),
		},
		{
			name: "as with lots of spaces",
			text: "true,    a    as      b; let c = d",
			want: makeShorthand("prefix", makeStatements(makeExpression("true", 0, "", 0), makeExpression("a", 9, "b", 22), makeLet("c", 29, "d", 33))),
		},
		{
			name: "key expr",
			text: "let odd, trackBy: trackByFn",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0), makeKeyExpr("trackBy", 9, "trackByFn", 18, "", 0))),
		},
		{
			name: "key expr with as local",
			text: "let odd, trackBy: trackByFn as fn",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0), makeKeyExpr("trackBy", 9, "trackByFn", 18, "fn", 31))),
		},
		{
			name: "key expr with as",
			text: "let odd, trackBy: trackByFn as fn; a as b",
			want: makeShorthand("prefix", makeStatements(makeLet("odd", 4, "", 0), makeKeyExpr("trackBy", 9, "trackByFn", 18, "fn", 31), makeExpression("a", 35, "b", 40))),
		},
		{
			name: "expression as at start",
			text: "odd as even",
			want: makeShorthand("prefix", makeStatements(makeExpression("odd", 0, "even", 7))),
		},
		{
			name: "double expression as with semicolon delimiter",
			text: "odd as even; even as odd;",
			want: makeShorthand("prefix", makeStatements(makeExpression("odd", 0, "even", 7), makeExpression("even", 13, "odd", 21))),
		},
		{
			name: "double expression as without delimiter",
			text: "odd as even even as odd;",
			want: makeShorthand("prefix", makeStatements(makeExpression("odd", 0, "even", 7), makeExpression("even", 12, "odd", 20))),
		},
		{
			name: "keyExpr with no delimiters",
			text: "x of y trackBy trackByFunc",
			want: makeShorthand("prefix", makeStatements(makeExpression("x", 0, "", 0), makeKeyExpr("of", 2, "y", 5, "", 0), makeKeyExpr("trackBy", 7, "trackByFunc", 15, "", 0))),
		},
		{
			name: "big expr",
			text: "true; false as by; true: j; true as b",
			want: makeShorthand("prefix", makeStatements(makeExpression("true", 0, "", 0), makeExpression("false", 6, "by", 15), makeKeyExpr("true", 19, "j", 25, "", 0), makeExpression("true", 28, "b", 36))),
		},
		{
			name:    "big expr error",
			text:    "true; false as by; true: j; true",
			wantErr: true,
		},
		{
			name: "ngFor one",
			text: "let item of items; index as i; trackBy: trackByFn",
			want: makeShorthand("prefix", makeStatements(makeLet("item", 4, "", 0), makeKeyExpr("of", 9, "items", 12, "", 0), makeExpression("index", 19, "i", 28), makeKeyExpr("trackBy", 31, "trackByFn", 40, "", 0))),
		},
		{
			name: "ngFor two",
			text: "let user of users; index as i; first as isFirst",
			want: makeShorthand("prefix", makeStatements(makeLet("user", 4, "", 0), makeKeyExpr("of", 9, "users", 12, "", 0), makeExpression("index", 19, "i", 28), makeExpression("first", 31, "isFirst", 40))),
		},
		{
			name: "ngFor three",
			text: "let user of [1, 2, 3] index as i; first as isFirst",
			want: makeShorthand("prefix", makeStatements(makeLet("user", 4, "", 0), makeKeyExpr("of", 9, "[1, 2, 3]", 12, "", 0), makeExpression("index", 22, "i", 31), makeExpression("first", 34, "isFirst", 43))),
		},
		{
			name: "ngFor four",
			text: "let user, of {1, 2, 3}; index as i; first as isFirst",
			want: makeShorthand("prefix", makeStatements(makeLet("user", 4, "", 0), makeKeyExpr("of", 10, "{1, 2, 3}", 13, "", 0), makeExpression("index", 24, "i", 33), makeExpression("first", 36, "isFirst", 45))),
		},
		{
			name: "unary operator good",
			text: "!false",
			want: makeShorthand("prefix", makeStatements(makeExpression("!false", 0, "", 0))),
		},
		{
			name:    "unary operator bad",
			text:    "!!!false",
			wantErr: true,
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
