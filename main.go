package main

import (
	"dataset_gen/ast/walk"
	"dataset_gen/parser"
	"dataset_gen/utils"
	"fmt"
	"io/fs"
	"log"
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
	fileContent []byte
	filePath    string
	pairs       []filePair
}

type output struct {
	prompt   string
	expected string
}

func main() {
	utils.InitQueries()

	root := "../../angular-tour-of-heroes"

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
		tsFile := file{content: parser.CStr2GoStr(state.fileContent), path: state.filePath}

		filePair := filePair{ts: tsFile, pug: pugFile}

		state.pairs = append(state.pairs, filePair)

		return state
	}

	state, err := utils.ParseFile(true, state.filePath, utils.TypeScript, state,
		func(root *sitter.Node, content []byte, state walkState) (walkState, error) {
			state.fileContent = content
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

		if string(prefix)+string(middle)+string(suffix) == string(content) {
			partial := partial{prefix: prefix, suffix: suffix, middle: middle}
			state.partials = append(state.partials, partial)
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

func generateOutputStrings(pair filePair) []output {
	outputs := make([]output, 0)

	base := "<filename>" + path.Base(pair.ts.path) + "\n" + pair.ts.content + "\n<filename>" + path.Base(pair.pug.path) + "\n"
	for _, partial := range pair.pugPartials {
		prompt := base + "<fim_suffix>" + parser.CStr2GoStr(partial.suffix) + "<fim_prefix>" + parser.CStr2GoStr(partial.prefix) + "<fim_middle>" + parser.CStr2GoStr(partial.middle)
		escaped := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(prompt, "\\", "\\\\"), "\"", "\\\""), "\n", "\\n")
		fmt.Println("{\"text\": \"" + escaped + "\"}")
	}

	return outputs
}
