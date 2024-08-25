package pug

import (
	"context"
	"regexp"
	"runtime/debug"
	"strings"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

var content []byte

var Depth = 0

var Logger = utils.GetLogger("pug-log.txt")

func visitJavascript(node *sitter.Node, state *State) {
	text := node.Content(content)

	nRanges := len(state.Ranges) - 1
	hasRanges := nRanges > 2
	proceedsEquals := state.Ranges[nRanges].NodeType == EQUALS

	isAttribute := hasRanges && proceedsEquals && state.Ranges[nRanges-1].NodeType == ATTRIBUTE_NAME
	isAngularAttribute := hasRanges && proceedsEquals && state.Ranges[nRanges-1].NodeType == ANGULAR_ATTRIBUTE_NAME

	isTemplateString := strings.ContainsRune(text, '`')
	isString, err := regexp.Match(`("(?:[^'\\]|\\.))|('(?:[^'\\]|\\.)+)`, []byte(text))

	if err != nil {
		panic(err) // if my regex is broken
	}

	r := getRange(node)

	quote := "'"
	if strings.Contains(text, "'") {
		quote = "\""
	}

	if nRanges < 1 || !(isAttribute || isAngularAttribute) {
		pushRangeSurround(state, text, r, quote, JAVASCRIPT)
		return
	}

	if isTemplateString {
		text = strings.ReplaceAll(text, "`", "")
		pushRange(state, `"$any('`, nil, nil)
		pushRange(state, text, &JAVASCRIPT, &r)
		pushRange(state, `')"`, nil, nil)
	} else if isString {
		pushRangeSurround(state, text, r, quote, JAVASCRIPT)
	} else if isAngularAttribute {
		escaped := strings.ReplaceAll(strings.ReplaceAll(text, `"`, `\"`), `'`, `\'`)
		pushRange(state, `"$any('`, nil, nil)
		pushRange(state, escaped, &JAVASCRIPT, &r)
		pushRange(state, `')"`, nil, nil)
	} else {
		pushRangeSurround(state, text, r, quote, JAVASCRIPT)
	}
}

func traverseTree(node *sitter.Node, state *State) {
	Depth = Depth + 1
	if Depth > 100 {
		Logger.Println("Overflow", string(debug.Stack()))
		return
	}

	nodeType := node.Type()
	ontent := node.Content(content)

	if ontent == "f" {
		return
	}

	if node.IsNamed() {
		if nodeType == "source_file" || nodeType == "children" || nodeType == "block_definition" || nodeType == "block_use" || nodeType == "each" {
			cursor := sitter.NewTreeCursor(node)
			cursor.GoToFirstChild()
			for {
				traverseTree(cursor.CurrentNode(), state)
				if !cursor.GoToNextSibling() { // TODO: replicate for all GoToNextSibling
					break
				}
			}
		} else if nodeType == "mixin_definition" {
			visitMixinDefinition(node, state)
		} else if nodeType == "iteration_variable" || nodeType == "iteration_iterator" {
			cursor := sitter.NewTreeCursor(node)
			cursor.GoToFirstChild()
			for {
				child := cursor.CurrentNode()
				if child.Type() == "javascript" {
					pushRange(state, "<script>return ", nil, nil)
					r := getRange(child)
					pushRange(state, child.Content(content), &JAVASCRIPT, &r)
					pushRange(state, ";</script>", nil, nil)
				} else {
					traverseTree(child, state)
				}

				if !cursor.GoToNextSibling() {
					break
				}
			}
		} else if nodeType == "script_block" {
			cursor := sitter.NewTreeCursor(node)
			cursor.GoToFirstChild()
			for cursor.GoToNextSibling() {
				child := cursor.CurrentNode()
				if child.Type() == "javascript" {
					pushRange(state, "<script>", nil, nil)
					r := getRange(child)
					pushRange(state, child.Content(content), &JAVASCRIPT, &r)
					pushRange(state, ";</script>", nil, nil)
				} else {
					traverseTree(child, state)
				}
			}
		} else if nodeType == "unbuffered_code" {
			visitUnbufferedCode(node, state)
		} else if nodeType == "buffered_code" || nodeType == "unescaped_buffered_code" {
			visitBufferedCode(node, state)
		} else if nodeType == "escaped_string_interpolation" {
			interpolationContent := node.Child(1)
			if interpolationContent != nil {
				text := interpolationContent.Content(content)
				pushRange(state, "<script>return ", nil, nil)
				r := getRange(interpolationContent)
				pushRange(state, text, &JAVASCRIPT, &r)
				pushRange(state, ";</script>", nil, nil)
			}
		} else if nodeType == "when" || nodeType == "case" {
			visitCaseWhen(node, state)
		} else if nodeType == "tag_interpolation" {
			visitTagInterpolation(node, state)
		} else if nodeType == "pipe" {
			visitPipe(node, state)
		} else if nodeType == "conditional" {
			visitConditional(node, state)
		} else if nodeType == "tag" || nodeType == "filter" {
			visitTag(node, state)
		} else if nodeType == "tag_name" || nodeType == "filter_name" {
			visitTagName(node, state)
		} else if nodeType == "attributes" {
			visitAttributes(node, state)
		} else if nodeType == "attribute_name" {
			visitAttributeName(node, state)
		} else if nodeType == "javascript" {
			visitJavascript(node, state)
		} else if nodeType == "quoted_attribute_value" {
			r := getRange(node)
			pushRange(state, node.Content(content), &ATTRIBUTE, &r)
		} else if nodeType == "content" {
			cursor := sitter.NewTreeCursor(node)
			cursor.GoToFirstChild()
			for {
				traverseTree(cursor.CurrentNode(), state)
				if !cursor.GoToNextSibling() {
					break
				}
			}
			// Always traverse the whole content after we've traversed the interpolation, so they
			// appear after in the conversion ranges
			r := getRange(node)
			pushRange(state, node.Content(content), &CONTENT, &r)
		} else if nodeType == "extends" || nodeType == "include" {
			visitExtendsInclude(node, state)
		} else if nodeType == "filename" {
			visitFilename(node, state)
		} else if node.IsError() {

			cursor := sitter.NewTreeCursor(node)
			cursor.GoToFirstChild()
			for cursor.GoToNextSibling() {
				traverseTree(cursor.CurrentNode(), state)

			}
		} else if nodeType == "keyword" || nodeType == "mixin_attributes" || nodeType == "comment" || nodeType == "block_name" {
			// No action
		} else {
			// Unhandled node type
		}
	}

	Depth = Depth - 1
}

func Parse(input string) (State, error) {
	content = []byte(input)

	parser := sitter.NewParser()
	parser.SetLanguage(utils.GetLanguage(utils.Pug))

	tree, err := parser.ParseCtx(context.TODO(), nil, []byte(input))
	if err != nil {
		return State{}, err
	}

	rootNode := tree.RootNode()

	state := State{
		HtmlText: "",
		PugText:  input,
		Ranges:   []Range{},
	}

	Depth = 0
	Logger.Println(input, string(debug.Stack()))

	traverseTree(rootNode, &state)

	state.HtmlText += "\n"

	lastEnd := uint32(0)
	if len(state.Ranges) > 0 {
		lastEnd = state.Ranges[len(state.Ranges)-1].PugEnd
	}

	state.Ranges = append(state.Ranges, Range{
		HtmlStart: uint32(len(state.HtmlText)) - 1,
		HtmlEnd:   uint32(len(state.HtmlText)),
		PugStart:  (lastEnd) + 1,
		PugEnd:    uint32(len(state.PugText)),
		NodeType:  EMPTY,
	})

	return state, nil
}
