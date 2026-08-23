package lsp

import (
	"strings"
	"ts_inspector/ast"
	"ts_inspector/config"
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/parser/tcb"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

type findDefinitionContext struct {
	*context
	locations []interfaces.Location
}

func findDefinition(baseContext *context) ([]interfaces.Location, error) {
	context := buildFindDefinitionContext(baseContext)

THING:
	for _, thing := range context.file.GetAvailableThings(context.state) {
		matchingSelectors, err := thing.SelectorsMatchingTag(context.ci.tagUnderCursor)
		if err != nil {
			return context.locations, err
		}

		for _, selector := range matchingSelectors {
			if handleFindDefinitionSelector(context, thing, selector) {
				continue THING
			}
		}
	}

	if config.GetConfig().TsGo.Enable {
		err := buildTsGoFindLocation(context)
		if err != nil {
			return context.locations, err
		}
	}

	return context.locations, nil
}

func buildDefinitionFindLocation(context *findDefinitionContext, definition parser.ClassedDefinition) bool {
	c := definition.Class.Snapshot()
	f := c.File.Snapshot()

	node := definition.Node
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = node
	}

	context.locations = append(context.locations, constructLocation(f.URI, c.Node, nameNode, f.Content))

	return true
}

func buildFindDefinitionContext(context *context) *findDefinitionContext {
	return &findDefinitionContext{context: context, locations: []interfaces.Location{}}
}

func buildTagFindLocation(context *findDefinitionContext, class *parser.Class) bool {
	c := class.Snapshot()
	f := c.File.Snapshot()

	context.locations = append(context.locations, constructLocation(f.URI, c.Node, c.NameNode, f.Content))

	return true
}

func buildTsGoFindLocation(context *findDefinitionContext) error {
	part, err := tcb.PugToTsLocation(context.state, context.file, context.ci.cursorOffset, context.ci.cursorOffset)
	if err != nil {
		return err
	}

	if part == nil {
		return nil
	}

	tcbUri, err := context.file.GetTcbUri()
	if err != nil {
		return err
	}

	symbol := context.state.GetTsGo().GetSymbolAtPosition(tcbUri, *part.TsStartOffset)
	if symbol == nil {
		return nil
	}

DECLARATION:
	for _, declarationHandle := range symbol.Declarations {
		pos, end, err := context.state.GetTsGo().GetNodePosition(declarationHandle)
		if err != nil {
			return err
		}

		node, err := declarationHandle.ExtractNode()
		if err != nil {
			return err
		}

		if strings.HasPrefix(node.Path, "bundled") {
			continue
		}

		declarationFile, found := context.state.GetFile(node.Path)
		if !found {
			continue
		}

		for _, class := range declarationFile.Snapshot().Classes {
			definition := class.GetDefinitionInRange(pos, end)
			if definition != nil {
				buildDefinitionFindLocation(context, *definition)
				continue DECLARATION
			}
		}

		// If I wasn't able to map to a definition, fallback to trying to convert the offset
		context.locations = append(context.locations, declarationFile.GetLocationForOffset(pos+1, end+1))
	}

	return nil
}

func constructLocation(uri string, classNode *sitter.Node, locationNode *sitter.Node, content string) interfaces.Location {
	startOffset, endOffset := interfaces.OffsetNodeByNode(locationNode, classNode)

	start := utils.GetPositionForOffset(content, startOffset)
	end := utils.GetPositionForOffset(content, endOffset)

	return interfaces.Location{Uri: uri, Range: utils.Range{Start: start, End: end}}
}

func handleFindDefinitionSelector(context *findDefinitionContext, thing *parser.Class, selector *ast.Selector) bool {
	handled := false

	if context.ci.isOnTagName {
		handled = handled || buildTagFindLocation(context, thing)
	}

	if context.ci.isOnAttrName {
		handled = handled || handleAttributeOfTag(context, thing, selector)
	}

	return handled
}

func handleAttributeOfTag(context *findDefinitionContext, thing *parser.Class, selector *ast.Selector) bool {
	attributeName := context.ci.attributeUnderCursor
	handled := false

	for _, definition := range thing.FilterAllDefinitions(func(def parser.ClassedDefinition) bool { return def.NameMatchesString(attributeName) }) {
		handled = buildDefinitionFindLocation(context, definition) || handled
	}

	if selector.MatchesAttribute(context.ci.attributeUnderCursor) { // component with `selector: '[formControl]`
		handled = buildTagFindLocation(context, thing) || handled
	}

	if !strings.HasPrefix(attributeName, "[") && !strings.HasSuffix(attributeName, "]") { // attribute with *ngIf
		handled = buildTagFindLocation(context, thing) || handled
	}

	return handled
}
