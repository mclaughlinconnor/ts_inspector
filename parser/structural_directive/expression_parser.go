package structuraldirective

import "ts_inspector/utils"

// Loosely based on https://github.com/mclaughlinconnor/tree-sitter-pug/blob/attrs/src/scanner.c

func ParseExpression(startIndex int, runeText []rune) (int, error) {
	i := startIndex
	paren := rune(0)
	parenStack := utils.NewStack[rune]()

	for i < len(runeText) {
		c := runeText[i]

		if parenStack.IsEmpty() {
			if isOpenParen(c) {
				parenStack.Push(switchBracket(c))
				i++
				continue
			}

			if !isOperator(c, paren == 0) && !isJsIdentifierChar(c) && c != ' ' {
				break
			}

			if c != ' ' {
				i++
				continue
			}

			// about to do lookaheads
			if i == len(runeText)-1 {
				break
			}

			prev := runeText[i-1]

			// Now, c can only be ' '

			// Exclude any trailing strings from the expression
			next := runeText[i+1]
			j := i
			for next == ' ' {
				j++
				next = runeText[j+1]
			}

			// !isOperator because "four+ five" won't have entered into the other isOperator behaviour below
			if !isOperator(prev, paren == 0) && isJsIdentifierChar(next) {
				break
			}

			if j == len(runeText)-1 {
				break
			} else {
				i = j
			}

			i++

			if isOperator(runeText[i], paren == 0) {
				i++ // current is now the one after next

				// Skip all of the trailing spaces. "x +___y"
				for runeText[i] == ' ' {
					i++
				}
			}

			continue
		}

		if isOpenParen(c) {
			parenStack.Push(switchBracket(c))
			i++
			continue
		}

		if *parenStack.Peek() == c {
			parenStack.Pop()
			i++
			continue
		}

		i++
	}

	return i, nil
}

func isOpenParen(c rune) bool {
	switch c {
	case '(':
		fallthrough
	case '[':
		fallthrough
	case '{':
		fallthrough
	case '\'':
		fallthrough
	case '"':
		return true
	}

	return false
}

func switchBracket(c rune) rune {
	switch c {
	case '(':
		return ')'
	case '[':
		return ']'
	case '{':
		return '}'
	case '\'':
		return '\''
	case '"':
		return '"'
	}

	return ' '
}

func isOperator(c rune, isRoot bool) bool {
	switch c {
	case ';':
		fallthrough
	case ',':
		return !isRoot
	case ':':
		fallthrough
	case '$':
		fallthrough
	case '&':
		fallthrough
	case '*':
		fallthrough
	case '+':
		fallthrough
	case '-':
		fallthrough
	case '.':
		fallthrough
	case '/':
		fallthrough
	case '<':
		fallthrough
	case '=':
		fallthrough
	case '>':
		fallthrough
	case '?':
		fallthrough
	case '^':
		fallthrough
	case '|':
		fallthrough
	case '!':
		return true
	}

	return false
}
