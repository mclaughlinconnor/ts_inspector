package structuraldirective

import (
	"fmt"
	"strings"
	"ts_inspector/utils"
	"unicode"
)

type ShorthandValue struct {
	Prefix     string
	Statements utils.HelpfulArray[*Statement]
}

type Statement struct {
	Expression *Expression
	KeyExp     *KeyExp
	Let        *Let
}

type Expression struct {
	Expression       string
	ExpressionOffset int
	Local            *string // as
	LocalOffset      int
}

type KeyExp struct {
	Key              string
	KeyOffset        int
	Expression       string
	ExpressionOffset int
	Local            *string
	LocalOffset      int
}

type Let struct {
	Export       *string
	ExportOffset int
	Local        string
	LocalOffset  int
}

func (shv *ShorthandValue) GetExpression() *Expression {
	if len(shv.Statements.Elements) == 0 {
		return nil
	}

	if !shv.Statements.Elements[0].HasExpression() {
		return nil
	}

	return shv.Statements.Elements[0].Expression
}

func (shv *ShorthandValue) GetKeyExprWithKey(queryKey string, needsPrefix bool) (bool, *KeyExp) {
	for _, s := range shv.Statements.Elements {
		if !s.HasKeyExp() {
			continue
		}

		keyExp := s.KeyExp

		if keyExp.Matches(shv, queryKey) {
			return true, s.KeyExp
		}
	}

	return false, nil
}

func (s *Statement) HasExpression() bool {
	return s.Expression != nil
}

func (s *Statement) HasKeyExp() bool {
	return s.KeyExp != nil
}

func (s *Statement) HasLet() bool {
	return s.Let != nil
}

func (l *Let) HasExport() bool {
	return l.Export != nil
}

func (k *KeyExp) GetFullName(shv *ShorthandValue) string {
	titleCasedKey := shv.Prefix + strings.ToUpper(k.Key[:1]) + k.Key[1:]
	return titleCasedKey
}

func (k *KeyExp) Matches(shv *ShorthandValue, queryKey string) bool {
	if shv.Prefix+k.Key == queryKey {
		return true
	}

	return queryKey == k.GetFullName(shv)
}

const (
	LexStateBase int = iota
	LexStateLet
	LexStateExpression
	LexStateAs
	LexStateKeyExp
)

// https://angular.dev/guide/directives/structural-directives#structural-directive-syntax-reference

func ParseShorthand(prefix string, text string) (*ShorthandValue, error) {
	shorthand := ShorthandValue{Prefix: prefix, Statements: utils.HelpfulArray[*Statement]{}}

	state := LexStateBase

	i := 0
	runeText := []rune(text)
LOOP:
	for i < len(runeText) {
		c := runeText[i]

		if i+1 == len(runeText) && (c == ',' || c == ';') {
			break
		}

		switch state {
		case LexStateBase:
			{
				if c == ' ' {
					i++
					continue LOOP
				}

				endIndex, err := ParseExpression(i, runeText)
				if err != nil {
					return nil, err
				}
				expressionText := string(runeText[i:endIndex])

				if expressionText == "let" {
					state = LexStateLet
					continue LOOP
				}

				// On the first statement, only expressions and let are valid
				if len(shorthand.Statements.Elements) == 0 {
					state = LexStateExpression
					continue LOOP
				}

				if !isHtmlIdentifierString(expressionText) {
					state = LexStateExpression
					continue LOOP
				}

				// consume to next token
				j := endIndex
				for {
					if j >= len(runeText)-1 {
						state = LexStateExpression
						continue LOOP
					}

					if runeText[j] != ' ' {
						break
					}

					j++
				}

				if runeText[j] == ':' {
					state = LexStateKeyExp
					continue LOOP
				}

				endIndex, err = ParseExpression(j, runeText)
				expressionText = string(runeText[j:endIndex])

				if expressionText == "as" {
					state = LexStateExpression
					continue LOOP
				}

				state = LexStateKeyExp
				continue LOOP
			}
		case LexStateLet:
			{
				if e := expectAndTakeLet(&i, runeText); e != nil {
					return nil, e
				}

				if e := expectAndTake(&i, runeText[i], ' '); e != nil {
					return nil, e
				}

				localOffset := i
				local, e := expectAndTakeIdentifier(&i, runeText, true)
				if e != nil {
					return nil, e
				}

				let := Let{Local: local, LocalOffset: localOffset}
				statement := Statement{Let: &let}
				shorthand.Statements.Add(&statement)

				if i == len(runeText) {
					state = LexStateBase
					continue LOOP
				}

				e = consumeSpaces(&i, runeText, false)
				if e != nil {
					return nil, e
				}

				if runeText[i] != '=' {
					state = LexStateBase
					consumeOptionalDelimiter(&i, runeText)
					continue LOOP
				}

				if e := expectAndTake(&i, runeText[i], '='); e != nil {
					return nil, e
				}

				e = consumeSpaces(&i, runeText, false)
				if e != nil {
					return nil, e
				}

				exportOffset := i
				export, e := expectAndTakeIdentifier(&i, runeText, true)
				if e != nil {
					return nil, e
				}

				let.Export = &export
				let.ExportOffset = exportOffset

				if i == len(runeText) {
					state = LexStateBase
					continue LOOP
				}

				consumeOptionalDelimiter(&i, runeText)
				state = LexStateBase
			}
		case LexStateExpression:
			{
				endIndex, err := ParseExpression(i, runeText)
				if err != nil {
					return nil, err
				}

				expressionText := runeText[i:endIndex]
				expression := Expression{Expression: string(expressionText), ExpressionOffset: i}
				statement := Statement{Expression: &expression}
				shorthand.Statements.Add(&statement)

				i = endIndex

				saveI := i

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					i = saveI
					return nil, err
				}

				if i >= len(runeText)-2 || runeText[i] != 'a' && runeText[i+1] != 's' {
					consumeOptionalDelimiter(&i, runeText)
					state = LexStateBase
					continue LOOP
				}

				if e := expectAndTakeAs(&i, runeText); e != nil {
					i = saveI
					return nil, e
				}

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				localOffset := i
				local, e := expectAndTakeIdentifier(&i, runeText, true)
				if e != nil {
					return nil, e
				}

				expression.Local = &local
				expression.LocalOffset = localOffset

				consumeOptionalDelimiter(&i, runeText)

				state = LexStateBase
			}
		case LexStateKeyExp:
			{
				keyOffset := i
				key, e := expectAndTakeIdentifier(&i, runeText, false)
				if e != nil {
					return nil, e
				}

				e = consumeSpaces(&i, runeText, false)
				if e != nil {
					return nil, e
				}

				// ':' is optional
				_ = expectAndTake(&i, runeText[i], ':')

				e = consumeSpaces(&i, runeText, false)
				if e != nil {
					return nil, e
				}

				endIndex, e := ParseExpression(i, runeText)
				if e != nil {
					return nil, e
				}

				expression := string(runeText[i:endIndex])

				keyExp := KeyExp{Key: key, KeyOffset: keyOffset, Expression: expression, ExpressionOffset: i}
				statement := Statement{KeyExp: &keyExp}
				shorthand.Statements.Add(&statement)

				i = endIndex + 1

				if i >= len(runeText) {
					state = LexStateBase
					consumeOptionalDelimiter(&i, runeText)
					continue
				}

				e = consumeSpaces(&i, runeText, false)
				if e != nil {
					return nil, e
				}

				if i+2 >= len(runeText) || runeText[i] != 'a' || runeText[i+1] != 's' || runeText[i+2] != ' ' {
					state = LexStateBase
					consumeOptionalDelimiter(&i, runeText)
					continue
				}

				if e := expectAndTakeAs(&i, runeText); e != nil {
					return nil, e
				}

				e = consumeSpaces(&i, runeText, true)
				if e != nil {
					return nil, e
				}

				localOffset := i
				local, e := expectAndTakeIdentifier(&i, runeText, true)
				if e != nil {
					return nil, e
				}

				keyExp.Local = &local
				keyExp.LocalOffset = localOffset

				consumeOptionalDelimiter(&i, runeText)
				state = LexStateBase
			}
		}
	}

	return &shorthand, nil
}

func consumeOptionalDelimiter(i *int, runeText []rune) {
	if (*i) >= len(runeText) {
		return
	}

	if runeText[*i] == ';' || runeText[*i] == ',' {
		(*i)++
	}
}

func consumeSpaces(i *int, runeText []rune, requireOne bool) error {
	if requireOne {
		if e := expectAndTake(i, runeText[*i], ' '); e != nil {
			return e
		}
	}

	for (*i) < len(runeText) && runeText[*i] == ' ' {
		(*i)++
	}

	return nil
}

func expectAndTake(i *int, c rune, expected rune) error {
	if c != expected {
		return fmt.Errorf("bad character: %q, expected %q", c, expected)
	}

	(*i)++

	return nil
}

func expectAndTakeAs(i *int, runeText []rune) error {
	if e := expectAndTake(i, runeText[*i], 'a'); e != nil {
		return e
	}
	if e := expectAndTake(i, runeText[*i], 's'); e != nil {
		return e
	}

	return nil
}

func expectAndTakeIdentifier(i *int, runeText []rune, jsIdentifier bool) (string, error) {
	index := *i

	c := runeText[index]

	if !unicode.IsLetter(c) && c != '_' { // The first char of an identifier can't be a number
		return "", fmt.Errorf("bad character: %q, expected identifier", c)
	}

	start := index
	index++

	var isIdentifier func(c rune) bool
	if jsIdentifier {
		isIdentifier = isJsIdentifierChar
	} else {
		isIdentifier = isHtmlIdentifierChar
	}

	for index < len(runeText) {
		if !isIdentifier(runeText[index]) {
			break
		}

		index++
	}

	*i = index

	return string(runeText[start:index]), nil
}

func expectAndTakeLet(i *int, runeText []rune) error {
	if e := expectAndTake(i, runeText[*i], 'l'); e != nil {
		return e
	}
	if e := expectAndTake(i, runeText[*i], 'e'); e != nil {
		return e
	}
	if e := expectAndTake(i, runeText[*i], 't'); e != nil {
		return e
	}

	return nil
}

func isHtmlIdentifierChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' || c == '-'
}

func isHtmlIdentifierString(s string) bool {
	for _, c := range s {
		if !isHtmlIdentifierChar(c) {
			return false
		}
	}

	return true
}

func isJsIdentifierChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_'
}
