package structuraldirective

// Loosely based on https://github.com/mclaughlinconnor/tree-sitter-pug/blob/attrs/src/scanner.c

func ParseExpression(startIndex int, runeText []rune) (int, error) {
	i := startIndex
	paren := rune(0)

	for i < len(runeText) {
		c := runeText[i]

		if paren == 0 {
			if isOpenParen(c) {
				paren = switchBracket(c)
				i++
				continue
			}

			if c == ',' || c == ';' {
				break
			}

			if c != ' ' {
				i++
				continue
			}

			// about to do lookaheads
			if i == len(runeText) {
				break
			}

			prev := runeText[i-1]

			// Now, c can only be ' '

			next := runeText[i+1]
			for next == ' ' {
				i++
				next = runeText[i+1]
			}

			// !isOperator because "four+ five" won't have entered into the other isOperator behaviour below
			if !isOperator(prev) && isJsIdentifierChar(next) || next == ';' || next == ',' {
				break
			}

			i++

			if isOperator(runeText[i]) {
				i++ // current is now the one after next

				// Skip all of the trailing spaces. "x +___y"
				for runeText[i] == ' ' {
					i++
				}
			}

			continue
		}

		if paren == c {
			paren = 0
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

func isOperator(c rune) bool {
	switch c {
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
	case ':':
		fallthrough
	case ';':
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
