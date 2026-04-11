package ast

import (
	"fmt"
	"slices"
	"strings"
	"ts_inspector/utils"
	"unicode"
)

type Selector struct {
	Attributes    []string
	NotAttributes []string
	NotTags       []string
	Tag           string
}

func (s *Selector) MatchesAttribute(attribute string) bool {
	strippedAttr, _ := utils.StripAngularFromAttribute(attribute)
	return slices.ContainsFunc(s.Attributes, func(a string) bool { sa, _ := utils.StripAngularFromAttribute(a); return sa == strippedAttr })
}

func (s *Selector) WithoutAttributes() *Selector {
	return &Selector{
		Attributes:    []string{},
		NotAttributes: s.NotAttributes,
		NotTags:       s.NotTags,
		Tag:           s.Tag,
	}
}

const (
	LexStateBase int = iota
	LexStateTag
	LexStateAttr
	LexStateNot
)

// Doesn't support CSS selectors `selector: ':not(.user)'`
func ParseSelector(text string) (*Selector, error) {
	selector := Selector{}

	state := LexStateBase

	buf := strings.Builder{}

	notState := false

	i := 0
	runeText := []rune(text)
	for i < len(runeText) {
		c := runeText[i]

		switch state {
		case LexStateBase:
			{
				if isIdentifierChar(c) {
					state = LexStateTag
					continue
				}

				switch c {
				case '[':
					{
						i++
						state = LexStateAttr
					}
				case ':':
					{
						i++
						if e := expectAndTake(&i, runeText[i], 'n'); e != nil {
							return nil, e
						}
						if e := expectAndTake(&i, runeText[i], 'o'); e != nil {
							return nil, e
						}
						if e := expectAndTake(&i, runeText[i], 't'); e != nil {
							return nil, e
						}
						state = LexStateNot
					}
				case '(':
					fallthrough
				case ')':
					{
						state = LexStateNot
					}
				default:
					{
						i++
					}
				}

			}
		case LexStateTag:
			{
				if isIdentifierChar(c) {
					buf.WriteRune(c)
					i++
					continue
				}

				if notState {
					selector.NotTags = append(selector.NotTags, buf.String())
				} else {
					selector.Tag = buf.String()
				}

				buf.Reset()
				state = LexStateBase
			}
		case LexStateNot:
			{
				switch c {
				case '(':
					notState = true
				case ')':
					notState = false
				}

				i++
				state = LexStateBase
			}
		case LexStateAttr:
			{
				if isIdentifierChar(c) {
					buf.WriteRune(c)
					i++
					continue
				}

				if c == ']' {
					if notState {
						selector.NotAttributes = append(selector.NotAttributes, buf.String())
					} else {
						selector.Attributes = append(selector.Attributes, buf.String())
					}

					buf.Reset()
					i++
					state = LexStateBase

					continue
				}

				badChar(c, "] or letter")
				i++
			}
		}
	}

	if buf.Len() == 0 {
		return &selector, nil
	}

	switch state {
	case LexStateAttr:
		{
			return nil, fmt.Errorf("Unfinished attribute")
		}
	case LexStateTag:
		{
			selector.Tag = buf.String()
			buf.Reset()
		}
	case LexStateNot:
		{
			return nil, fmt.Errorf("Unfinished not")
		}
	}

	return &selector, nil
}

func ParseSelectors(text string) (*[]*Selector, error) {
	splits := strings.Split(text, ",")

	selectors := make([]*Selector, len(splits))
	for i, s := range splits {
		selector, err := ParseSelector(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}

		selectors[i] = selector
	}

	return &selectors, nil
}

func expectAndTake(i *int, c rune, expected rune) error {
	(*i)++

	if c != expected {
		return badChar(c, string(expected))
	}

	return nil
}

func badChar(actual rune, expected string) error {
	return fmt.Errorf("Bad character: '" + string(actual) + "', expected " + expected)
}

func isIdentifierChar(c rune) bool {
	return unicode.IsLetter(c) || c == '-' || c == '_'
}
