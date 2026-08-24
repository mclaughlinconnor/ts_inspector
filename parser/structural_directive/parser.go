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

func (shv *ShorthandValue) GetKeyExprWithKey(queryKey string) *KeyExp {
	for _, s := range shv.Statements.Elements {
		if !s.HasKeyExp() {
			continue
		}

		keyExp := s.KeyExp

		if keyExp.Matches(shv, queryKey) {
			return s.KeyExp
		}
	}

	return nil
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

				endIndex := i

				unaryOperator, err := consumeOptionalUnaryOperator(&endIndex, runeText)
				if err != nil {
					return nil, err
				}

				if unaryOperator != "" {
					state = LexStateExpression
					continue LOOP
				}

				expressionText, err := expectAndTakeIdentifier(&endIndex, runeText, true)
				if err == nil && expressionText == "let" {
					state = LexStateLet
					continue LOOP
				}

				// On the first statement, only expressions and let are valid
				if len(shorthand.Statements.Elements) == 0 {
					state = LexStateExpression
					continue LOOP
				}

				// consume to next token
				j := endIndex
				for {
					if j >= len(runeText)-1 {
						state = LexStateKeyExp
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
				if err != nil {
					return nil, err
				}

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
				if err := expectAndTakeLet(&i, runeText); err != nil {
					return nil, err
				}

				if err := expectAndTake(&i, runeText, ' '); err != nil {
					return nil, err
				}

				localOffset := i
				local, err := expectAndTakeIdentifier(&i, runeText, true)
				if err != nil {
					return nil, err
				}

				let := Let{Local: local, LocalOffset: localOffset}
				statement := Statement{Let: &let}
				shorthand.Statements.Add(&statement)

				if i == len(runeText) {
					state = LexStateBase
					continue LOOP
				}

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				if runeText[i] != '=' {
					state = LexStateBase
					consumeOptionalDelimiter(&i, runeText)
					continue LOOP
				}

				if err := expectAndTake(&i, runeText, '='); err != nil {
					return nil, err
				}

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				exportOffset := i
				export, err := expectAndTakeIdentifier(&i, runeText, true)
				if err != nil {
					return nil, err
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
				// NOTE: After the first statement, the first part of the expression must be an identifier, not a generic expression
				// I.e., `true as a, true as b` is valid and `[a, b, c] as d, true as a` is valid, but `true as a, [a, b, c] as d` is invalid
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

				if err := expectAndTakeAs(&i, runeText); err != nil {
					i = saveI
					return nil, err
				}

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				localOffset := i
				local, err := expectAndTakeIdentifier(&i, runeText, true)
				if err != nil {
					return nil, err
				}

				expression.Local = &local
				expression.LocalOffset = localOffset

				consumeOptionalDelimiter(&i, runeText)

				state = LexStateBase
			}
		case LexStateKeyExp:
			{
				keyOffset := i
				key, err := expectAndTakeIdentifier(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				err = expectNotEof(&i, runeText)
				if err != nil {
					return nil, err
				}

				// ':' is optional
				take(&i, runeText, ':')

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				endIndex, err := ParseExpression(i, runeText)
				if err != nil {
					return nil, err
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

				err = consumeSpaces(&i, runeText, false)
				if err != nil {
					return nil, err
				}

				if i+2 >= len(runeText) || runeText[i] != 'a' || runeText[i+1] != 's' || runeText[i+2] != ' ' {
					state = LexStateBase
					consumeOptionalDelimiter(&i, runeText)
					continue
				}

				if err := expectAndTakeAs(&i, runeText); err != nil {
					return nil, err
				}

				err = consumeSpaces(&i, runeText, true)
				if err != nil {
					return nil, err
				}

				localOffset := i
				local, err := expectAndTakeIdentifier(&i, runeText, true)
				if err != nil {
					return nil, err
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

func consumeOptionalUnaryOperator(i *int, runeText []rune) (string, error) {
	start := *i
	index := *i

	c := runeText[index]
	howMany := 0

LOOP:
	for {

		switch c {
		case '!':
			fallthrough
		case '+':
			fallthrough
		case '-':
			{
				howMany++
				index++
				c = runeText[index]
			}
		default:
			break LOOP
		}

		if howMany >= 2 {
			return "", fmt.Errorf("bad character: %q, may not be chained more than twice", c)
		}
	}

	*i = index

	return string(runeText[start:index]), nil
}

func consumeSpaces(i *int, runeText []rune, requireOne bool) error {
	if requireOne {
		if err := expectAndTake(i, runeText, ' '); err != nil {
			return err
		}
	}

	for (*i) < len(runeText) && runeText[*i] == ' ' {
		(*i)++
	}

	return nil
}

func expectAndTake(i *int, runeText []rune, expected rune) error {
	if (*i) >= len(runeText) {
		return fmt.Errorf("bad eof, expected %q", expected)
	}

	c := runeText[*i]

	if c != expected {
		return fmt.Errorf("bad character: %q, expected %q", c, expected)
	}

	(*i)++

	return nil
}

func expectAndTakeAs(i *int, runeText []rune) error {
	if err := expectAndTake(i, runeText, 'a'); err != nil {
		return err
	}
	if err := expectAndTake(i, runeText, 's'); err != nil {
		return err
	}

	return nil
}

func expectAndTakeIdentifier(i *int, runeText []rune, jsIdentifier bool) (string, error) {
	index := *i

	c := runeText[index]

	var isIdentifier func(c rune) bool
	if jsIdentifier {
		isIdentifier = isJsIdentifierChar
	} else {
		isIdentifier = isHtmlIdentifierChar
	}

	if !isIdentifier(c) || unicode.IsNumber(c) { // The first char of an identifier can't be a number
		return "", fmt.Errorf("bad character: %q, expected identifier", c)
	}

	start := index
	index++

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
	if err := expectAndTake(i, runeText, 'l'); err != nil {
		return err
	}
	if err := expectAndTake(i, runeText, 'e'); err != nil {
		return err
	}
	if err := expectAndTake(i, runeText, 't'); err != nil {
		return err
	}

	return nil
}

func expectNotEof(i *int, runeText []rune) error {
	if (*i) >= len(runeText) {
		return fmt.Errorf("bad eof")
	}

	return nil
}

func isHtmlIdentifierChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' || c == '-'
}

func isJsIdentifierChar(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' || c == '$'
}

func take(i *int, runeText []rune, expected rune) {
	c := runeText[*i]
	if c != expected {
		return
	}

	(*i)++
}
