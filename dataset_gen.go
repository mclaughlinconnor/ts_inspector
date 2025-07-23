package main

import (
	"dataset_gen/ast/walk"
	"dataset_gen/parser"
	"dataset_gen/utils"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"path"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type partial struct {
	prefix []byte
	suffix []byte
	middle []byte
}

type cutWalkState struct {
	partials []partial
}

type file struct {
	content string
	path    string
}

type filePair struct {
	ts          file
	pug         file
	pugPartials []partial
}

type walkState struct {
	classContent []byte
	fileContent  []byte
	filePath     string
	pairs        []filePair
}

type output struct {
	prompt   string
	expected string
}

var model = "qwen" // "mellum" or "qwen"
var includeFullText = true

var filename_token = ""
var fim_suffix_token = ""
var fim_prefix_token = ""
var fim_middle_token = ""

func dataset_gen() {
	utils.InitQueries()

	if model == "mellum" {
		filename_token = "<filename>"
		fim_suffix_token = "<fim_suffix>"
		fim_prefix_token = "<fim_prefix>"
		fim_middle_token = "<fim_middle>"
	} else if model == "qwen" {
		filename_token = "<|file_sep|>"
		fim_suffix_token = "<|fim_suffix|>"
		fim_prefix_token = "<|fim_prefix|>"
		fim_middle_token = "<|fim_middle|>"
	}

	root := "../../development"

	state := walkState{fileContent: []byte(""), pairs: make([]filePair, 0)}

	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if info.IsDir() && (info.Name() == "node_modules" || strings.HasPrefix(info.Name(), ".")) {
			return filepath.SkipDir
		}

		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		if filepath.Ext(path) != ".ts" {
			return nil
		}

		state.filePath = path
		state = walkTypeScript(state)

		return nil
	})

	for _, pair := range state.pairs {
		_, err := utils.ParseFile(true, pair.pug.path, utils.Pug, state,
			func(root *sitter.Node, content []byte, state walkState) (walkState, error) {
				partials := createPartialsFromRoot(root, content)
				pair.pugPartials = partials.partials

				generateOutputStrings(pair)

				base := filename_token + path.Base(pair.ts.path) + "\n" + pair.ts.content + "\n" + filename_token + path.Base(pair.pug.path) + "\n"
				fmt.Println("{\"text\": \"" + escape(base) + escape(pair.pug.content) + "\"}")

				return state, nil
			})

		if err != nil {
			log.Println(err)
		}
	}
}

func walkTypeScript(state walkState) walkState {
	funcMap := walk.NewVisitorFuncsMap[walkState]()
	funcMap["export_statement"] = func(node *sitter.Node, state walkState, indexInParent int, funcMap walk.VisitorFuncMap[walkState]) walkState {
		decorator := node.ChildByFieldName("decorator")
		if decorator == nil {
			return state
		}

		for i := range decorator.NamedChildCount() {
			index := int(i)
			state = walk.VisitNode(decorator.NamedChild(index), state, index, funcMap)
		}

		return state
	}

	funcMap["call_expression"] = func(node *sitter.Node, state walkState, indexInParent int, funcMap walk.VisitorFuncMap[walkState]) walkState {
		functionName := node.ChildByFieldName("function")
		if functionName == nil {
			return state
		}

		if functionName.Content([]byte(state.fileContent)) != "Component" {
			return state
		}

		arguments := node.ChildByFieldName("arguments")
		if arguments == nil {
			return state
		}

		configArgument := arguments.NamedChild(0)
		if configArgument == nil {
			return state
		}

		for i := range configArgument.NamedChildCount() {
			index := int(i)
			state = walk.VisitNode(configArgument.NamedChild(index), state, index, funcMap)
		}

		return state
	}

	funcMap["pair"] = func(node *sitter.Node, state walkState, indexInParent int, funcMap walk.VisitorFuncMap[walkState]) walkState {
		key := node.ChildByFieldName("key")
		if key == nil {
			return state
		}

		if key.Content([]byte(state.fileContent)) != "templateUrl" {
			return state
		}

		valueStringFragment := node.ChildByFieldName("value")
		if valueStringFragment == nil {
			return state
		}

		value := valueStringFragment.NamedChild(0)
		if value == nil {
			return state
		}

		templateFilepath := value.Content([]byte(state.fileContent))

		if path.Ext(templateFilepath) != ".pug" {
			return state
		}

		fullTemplateFilepath := path.Join(filepath.Dir(state.filePath), templateFilepath)

		pugContent, err := utils.ReadFile(fullTemplateFilepath)
		if err != nil {
			log.Println(err)
			return state
		}

		pugFile := file{content: parser.CStr2GoStr(pugContent), path: fullTemplateFilepath}
		tsFile := file{content: parser.CStr2GoStr(state.classContent), path: state.filePath}

		filePair := filePair{ts: tsFile, pug: pugFile}

		state.pairs = append(state.pairs, filePair)

		return state
	}

	state, err := utils.ParseFile(true, state.filePath, utils.TypeScript, state,
		func(root *sitter.Node, content []byte, state walkState) (walkState, error) {
			state.fileContent = content

			state.classContent = []byte(extractSignatures(root, []byte(content)))

			state = walk.Walk(root, state, funcMap)

			return state, nil
		})

	if err != nil {
		log.Println(err)
	}

	return state
}

func createPartialsFromRoot(root *sitter.Node, content []byte) cutWalkState {
	funcMap := walk.NewVisitorFuncsMap[cutWalkState]()

	extractor := func(node *sitter.Node, state cutWalkState, indexInParent int, funcMap walk.VisitorFuncMap[cutWalkState]) cutWalkState {

		fileStart := root.StartByte()
		fileEnd := root.EndByte()

		middleStart := node.StartByte()
		middleEnd := node.EndByte()

		prefix := content[fileStart:middleStart]
		middle := content[middleStart:middleEnd]
		suffix := content[middleEnd:fileEnd]

		if string(prefix)+string(middle)+string(suffix) != string(content) {
			for i := range node.NamedChildCount() {
				index := int(i)
				state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
			}

			return state
		}

		p := partial{prefix: prefix, suffix: suffix, middle: middle}
		state.partials = append(state.partials, p)

		strMiddle := string(middle)

		middleLength := len(strMiddle)

		if middleLength > 5 {
			midwayPoint := int(middleLength / 2)

			startRandomOffsetOne := rand.Intn(midwayPoint)
			startRandomOffsetTwo := rand.Intn(midwayPoint)

			endRandomOffsetOne := rand.Intn(midwayPoint)
			endRandomOffsetTwo := rand.Intn(midwayPoint)

			partialWithPartial := func(midStart int, midEnd int) {
				strContent := string(content)
				pre := substring(strContent, int(fileStart), int(midStart))
				mid := substring(strContent, int(midStart), int(midEnd))
				suf := substring(strContent, int(midEnd), int(fileEnd))

				state.partials = append(state.partials, partial{prefix: []byte(pre), suffix: []byte(suf), middle: []byte(mid)})
			}

			partialWithPartial(int(middleStart)+startRandomOffsetOne, int(middleEnd))
			partialWithPartial(int(middleStart), int(middleEnd)-endRandomOffsetOne)
			partialWithPartial(int(middleStart)+startRandomOffsetTwo, int(middleEnd)-endRandomOffsetTwo)
		}

		if strings.Contains(strMiddle, "\n") {
			lines := strings.Split(strMiddle, "\n")
			for i := range lines {
				beforeLines := make([]string, 0)
				theLine := ""

				for j, line := range lines {
					if j < i {
						beforeLines = append(beforeLines, line)
					}
					if j == i {
						theLine = line
						break
					}
				}

				if len(theLine) == 0 {
					continue
				}

				newPrefix := string(prefix) + "\n" + strings.Join(beforeLines, "\n")
				p := partial{prefix: []byte(newPrefix), suffix: suffix, middle: []byte(theLine)}
				state.partials = append(state.partials, p)

				if len(theLine) <= 5 {
					continue
				}

				randomOffset := uint32(rand.Intn(len(theLine) / 2))
				p.prefix = []byte(newPrefix + theLine[0:randomOffset])
				p.middle = []byte(theLine[randomOffset:])
				state.partials = append(state.partials, p)
			}
		}

		for i := range node.NamedChildCount() {
			index := int(i)
			state = walk.VisitNode(node.NamedChild(index), state, index, funcMap)
		}

		return state
	}

	funcMap["tag"] = extractor
	funcMap["attributes"] = extractor
	funcMap["attribute"] = extractor
	funcMap["attribute_value"] = extractor

	state := cutWalkState{partials: make([]partial, 0)}

	state = walk.Walk(root, state, funcMap)

	return state
}

func escape(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(text, "\\", "\\\\"), "\"", "\\\""), "\n", "\\n")
}

func generateOutputStrings(pair filePair) []output {
	outputs := make([]output, 0)

	base := filename_token + path.Base(pair.ts.path) + "\n" + pair.ts.content + "\n" + filename_token + path.Base(pair.pug.path) + "\n"
	for _, partial := range pair.pugPartials {
		sft_text := ""
		prompt := ""
		output := ""
		if model == "mellum" {
			prompt = base + fim_suffix_token + parser.CStr2GoStr(partial.suffix) + fim_prefix_token + parser.CStr2GoStr(partial.prefix) + fim_middle_token
		} else if model == "qwen" {
			prompt = base + fim_prefix_token + parser.CStr2GoStr(partial.prefix) + fim_suffix_token + parser.CStr2GoStr(partial.suffix) + fim_middle_token
		}

		output = parser.CStr2GoStr(partial.middle)
		sft_text = prompt + output

		fmt.Println("{\"text\": \"" + escape(sft_text) + "\", \"answer\": \"" + escape(output) + "\", \"prompt\": \"" + escape(prompt) + "\"}")
	}

	return outputs
}

type classExtractState struct {
	body string
}

func extractSignatures(root *sitter.Node, content []byte) string {
	funcMap := walk.NewVisitorFuncsMap[classExtractState]()

	verbatim := func(node *sitter.Node, state classExtractState, indexInParent int, funcMap walk.VisitorFuncMap[classExtractState]) classExtractState {
		state.body = state.body + node.Content(content) + "\n"
		return state
	}

	comment := func(node *sitter.Node, state classExtractState, indexInParent int, funcMap walk.VisitorFuncMap[classExtractState]) classExtractState {
		sibling := node.NextSibling()

		if sibling == nil || (sibling.Type() != "public_field_definition" && sibling.Type() != "method_definition") {
			state.body = state.body + node.Content(content) + "\n"
			return state
		}

		for childIndex := range sibling.NamedChildCount() {
			childNode := sibling.NamedChild(int(childIndex))
			if childNode.Type() == "accessibility_modifier" && childNode.Content(content) == "private" {
				return state
			}
		}

		state.body = state.body + node.Content(content) + "\n"
		return state
	}

	skipPrivate := func(node *sitter.Node, state classExtractState, indexInParent int, funcMap walk.VisitorFuncMap[classExtractState]) classExtractState {
		body := state.body
		for childIndex := range node.ChildCount() {
			childNode := node.Child(int(childIndex))
			childContent := childNode.Content(content)
			if childNode.Type() == "accessibility_modifier" && childContent == "private" {
				return state
			}

			body = body + childContent + " "
		}

		state.body = body + " \n"
		return state
	}

	skipPrivateAndBody := func(node *sitter.Node, state classExtractState, indexInParent int, funcMap walk.VisitorFuncMap[classExtractState]) classExtractState {
		body := state.body
		for childIndex := range node.ChildCount() {
			childNode := node.Child(int(childIndex))
			childContent := childNode.Content(content)
			if childNode.Type() == "accessibility_modifier" && childContent == "private" {
				return state
			}

			if node.FieldNameForChild(int(childIndex)) == "body" {
				continue
			}

			body = body + childContent + " "
		}

		state.body = body + " \n"
		return state
	}

	classVisitor := func(node *sitter.Node, state classExtractState, indexInParent int, funcMap walk.VisitorFuncMap[classExtractState]) classExtractState {
		for childIndex := range node.ChildCount() {
			if node.FieldNameForChild(int(childIndex)) == "body" {
				state.body = state.body + "{\n"
				state = walk.VisitNode(node.Child(int(childIndex)), state, int(childIndex), funcMap)

				continue
			}

			state.body = state.body + node.Child(int(childIndex)).Content(content) + " "
		}

		return state
	}

	funcMap["decorator"] = verbatim
	funcMap["enum_declaration"] = verbatim
	funcMap["function_declaration"] = verbatim
	funcMap["interface_declaration"] = verbatim
	funcMap["lexical_declaration"] = verbatim
	funcMap["type_alias_declaration"] = verbatim

	funcMap["public_field_definition"] = skipPrivate
	funcMap["method_definition"] = skipPrivateAndBody

	funcMap["class_declaration"] = classVisitor

	funcMap["comment"] = comment

	state := classExtractState{body: ""}

	state = walk.Walk(root, state, funcMap)

	return state.body
}

// https://stackoverflow.com/a/38537764
func substring(s string, start int, end int) string {
	start_str_idx := 0
	i := 0
	for j := range s {
		if i == start {
			start_str_idx = j
		}
		if i == end {
			return s[start_str_idx:j]
		}
		i++
	}
	return s[start_str_idx:]
}
