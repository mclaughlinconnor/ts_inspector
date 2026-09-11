package reviewindexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"ts_inspector/ast/walk"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type Metadata struct {
	Category string // probably a union
	FilePath string
	LineKind string // probably a union
	NewLine  int
	OldLine  int
	Severity string // probably a union
}

type Content struct {
	Agent   string
	Summary string
}

const FINDING_PATH = ".mr-deep-review/synthesised"

func Index(rootPath string) ([]Finding, error) {
	findings := []Finding{}

	paths, err := listReviewYamls(filepath.Join(rootPath, FINDING_PATH))
	if err != nil {
		return findings, err
	}

	for _, path := range paths {
		finding, err := parseFromPath(rootPath, path)
		if err != nil {
			return findings, err
		}

		findings = append(findings, finding)
	}

	return findings, nil
}

func listReviewYamls(rootPath string) ([]string, error) {
	fileInfo, err := os.Stat(rootPath)
	if err != nil {
		return []string{}, err
	}

	if !fileInfo.IsDir() {
		return []string{}, fmt.Errorf("%v is not a directory", rootPath)
	}

	findings := []string{}

	files, err := os.ReadDir(FINDING_PATH)
	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		findings = append(findings, filepath.Join(rootPath, file.Name()))
	}

	return findings, nil
}

func parseReviewYaml(content []byte, rootPath string) (Metadata, error) {
	root, err := utils.ParseText(content, utils.Yaml)
	if err != nil {
		return Metadata{}, err
	}

	cursor := sitter.NewTreeCursor(root)

	err = goToNextNamedSiblingOfTypeAndChild(cursor, "stream")
	if err != nil {
		return Metadata{}, err
	}

	err = goToNextNamedSiblingOfTypeAndChild(cursor, "document")
	if err != nil {
		return Metadata{}, err
	}

	err = goToNextNamedSiblingOfTypeAndChild(cursor, "block_node")
	if err != nil {
		return Metadata{}, err
	}

	err = goToNextNamedSiblingOfTypeAndChild(cursor, "block_mapping")
	if err != nil {
		return Metadata{}, err
	}

	metadata := Metadata{}

	for cursor.CurrentNode().Type() == "block_mapping_pair" {
		err = handleYamlKv(cursor.CurrentNode(), content, &metadata, rootPath)
		if err != nil {
			return Metadata{}, err
		}

		// move off of the current node
		if !cursor.GoToNextSibling() {
			break
		}

		for !cursor.CurrentNode().IsNamed() {
			if !cursor.GoToNextSibling() {
				break
			}
		}
	}

	return metadata, nil
}

func handleYamlKv(node *sitter.Node, content []byte, metadata *Metadata, rootPath string) error {
	key := node.ChildByFieldName("key")
	if key == nil {
		return fmt.Errorf("Invalid yaml format: missing key")
	}

	value := node.ChildByFieldName("value")
	if value == nil {
		return fmt.Errorf("Invalid yaml format: missing value")
	}

	keyContent := key.Content(content)
	valueContent := value.Content(content)

	switch keyContent {
	case "filePath":
		metadata.FilePath = filepath.Join(rootPath, valueContent)
	case "lineKind":
		metadata.LineKind = valueContent
	case "newLine":
		number, err := strconv.Atoi(valueContent)
		if err != nil {
			return err
		}
		metadata.NewLine = number
	case "oldLine":
		number, err := strconv.Atoi(valueContent)
		if err != nil {
			return err
		}
		metadata.OldLine = number
	case "category":
		metadata.Category = valueContent
	case "severity":
		metadata.Severity = valueContent
	}

	return nil
}

func parseReviewMarkdown(root *sitter.Node, content []byte) (Content, error) {
	firstParagraph := root.NamedChild(0)
	if firstParagraph == nil {
		return Content{}, fmt.Errorf("Invalid markdown: unexpected EOF")
	}

	if firstParagraph.Type() != "paragraph" {
		return Content{}, fmt.Errorf("Invalid markdown: unexpected %v, expected 'paragraph'", firstParagraph.Type())
	}

	findingContent := Content{}
	findingContent.Summary = strings.ReplaceAll(string(firstParagraph.Content(content)), "\n", " ")

	agentParagraph := root.NamedChild(int(root.NamedChildCount() - 1))
	agentParagraphContent := agentParagraph.Content(content)

	agentPrefix := "**Agent**: "
	for !strings.HasPrefix(agentParagraphContent, agentPrefix) {
		agentParagraph = agentParagraph.PrevNamedSibling()
		agentParagraphContent = agentParagraph.Content(content)
	}

	agent := agentParagraphContent[len(agentPrefix):]
	findingContent.Agent = strings.TrimSpace(agent)

	return findingContent, nil
}

type Finding struct {
	FilePath string
	Metadata Metadata
	Content  Content
}

func parseFromPath(rootPath string, path string) (Finding, error) {
	root, content, err := utils.ParseTextFromPath(path, utils.Markdown)
	if err != nil {
		return Finding{}, err
	}

	funcMap := walk.NewVisitorFuncsMap[Finding]()
	funcMap["document"] = func(node *sitter.Node, state Finding, indexInParent int, visitorFuncMap walk.VisitorFuncMap[Finding]) (Finding, error) {
		return walk.VisitNamedChildren(node, state, visitorFuncMap, false)
	}

	funcMap["minus_metadata"] = func(node *sitter.Node, state Finding, indexInParent int, visitorFuncMap walk.VisitorFuncMap[Finding]) (Finding, error) {
		metadata, err := parseReviewYaml([]byte(node.Content(content)), rootPath)
		if err != nil {
			return state, err
		}

		state.Metadata = metadata

		return state, nil
	}

	funcMap["section"] = func(node *sitter.Node, state Finding, indexInParent int, visitorFuncMap walk.VisitorFuncMap[Finding]) (Finding, error) {
		content, err := parseReviewMarkdown(node, content)
		if err != nil {
			return state, err
		}

		state.Content = content

		return state, nil
	}

	return walk.Walk(root, Finding{FilePath: path}, funcMap, utils.GetLanguage(utils.Markdown), true)
}

func goToNextNamedSiblingOfTypeAndChild(cursor *sitter.TreeCursor, nodeType string) error {
	for !cursor.CurrentNode().IsNamed() {
		if !cursor.GoToNextSibling() {
			return fmt.Errorf("Invalid yaml format: unexpected EOF")
		}
	}

	if cursor.CurrentNode().Type() != nodeType {
		return fmt.Errorf("Invalid yaml format: missing '%v' type node", nodeType)
	}

	if !cursor.GoToFirstChild() {
		return fmt.Errorf("Invalid yaml format: unexpectedly missing child")
	}

	return nil
}
