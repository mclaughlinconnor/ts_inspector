package parser

import (
	"bytes"
	"strings"
	"ts_inspector/ast"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

func IsAngularDecorator(name string) bool {
	_, found := angularDecorators[name]

	return found
}

var angularDecorators = map[string]bool{
	"Attribute":       true,
	"Component":       true,
	"ContentChild":    true,
	"ContentChildren": true,
	"Directive":       true,
	"Host":            true,
	"HostBinding":     true,
	"HostListener":    true,
	// Silence the "Angular property should be public" error on @Inject
	// TODO: find a better solution for this
	// "Inject":          true,
	"Injectable":   true,
	"Input":        true,
	"NgModule":     true,
	"Optional":     true,
	"Output":       true,
	"Pipe":         true,
	"Self":         true,
	"SkipSelf":     true,
	"ViewChild":    true,
	"ViewChildren": true,
}

func IsAngularFunction(name string) bool {
	_, found := angularFunctions[name]

	return found
}

var angularFunctions = map[string]bool{
	"ngAfterContentChecked": true,
	"ngAfterContentInit":    true,
	"ngAfterViewChecked":    true,
	"ngAfterViewInit":       true,
	"ngDoCheck":             true,
	"ngOnChanges":           true,
	"ngOnDestroy":           true,
	"ngOnInit":              true,
	"constructor":           true,
	"writeValue":            true,
	"normaliseWriteValue":   true,
}

func CStr2GoStr(b []byte) string {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		i = len(b)
	}

	return string(b[:i])
}

func FilenameFromUri(uri string) string {
	return strings.TrimPrefix(uri, `file://`)
}

func UriFromFilename(filename string) string {
	return `file://` + filename
}

func FindDefinition(state *State, file *File, cursorOffset uint32) []interfaces.Location {
	locations := make([]interfaces.Location, 0)

	tagName, cursorOnTagName := ast.GetTagNameAtOffset(file.Snapshot().Content, cursorOffset)

	var tagUnderCursor ast.Tag
	attributeName, cursorOnAttributeName := ast.GetAttributeNameAtOffset(file.Snapshot().Content, cursorOffset)
	if cursorOnAttributeName {
		tagUnderCursor, _ = ast.GetTagAtOffset(file.Snapshot().Content, cursorOffset)
	}

	for _, c := range file.Snapshot().Classes {
		if !c.HasComponent() {
			continue
		}

		things := c.Snapshot().Angular.Component.GetAvailableThings(state)
		for _, thing := range things {
			selectors := []string{}
			if thing.HasComponent() {
				selectors = append(selectors, thing.Snapshot().Angular.Component.Selectors...)
			}

			if thing.HasDirective() {
				selectors = append(selectors, thing.Snapshot().Angular.Directive.Selectors...)
			}

			for _, selector := range selectors {
				if cursorOnTagName && selector == tagName {
					locations = handleDefinitionOfTag(locations, thing)
				}

				if cursorOnAttributeName {
					matches, parsed := tagUnderCursor.MatchesSelector(selector)
					if !matches {
						continue
					}

					locations = handleAttributeOfTag(locations, thing, attributeName, selector)

					if parsed.MatchesAttribute(attributeName) { // component with `selector: '[formControl]`
						locations = handleDefinitionOfTag(locations, thing)
					}
				}
			}
		}
	}

	return locations
}

func handleAttributeOfTag(locations []interfaces.Location, class *Class, attributeName string, selector string) []interfaces.Location {
	for _, definition := range class.FilterAllDefinitions(func(def ClassedDefinition) bool { return def.NameMatchesString(attributeName) }) {
		c := definition.Class.Snapshot()
		cOffset := c.Node.StartByte()
		file := c.File.Snapshot()
		cContent := file.Content

		node := definition.Node

		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			nameNode = node
		}

		start := utils.GetPositionForOffset(cContent, cOffset+nameNode.StartByte())
		end := utils.GetPositionForOffset(cContent, cOffset+nameNode.StartByte())

		locations = append(locations, interfaces.Location{Uri: file.URI, Range: utils.Range{Start: start, End: end}})
	}

	if !strings.HasPrefix(attributeName, "[") && !strings.HasSuffix(attributeName, "]") {
		valid, _, attrName := ast.ExtractTagNameAndAttrFromSelector(selector)
		if valid && attrName == attributeName {
			c := class.Snapshot()
			f := c.File.Snapshot()

			startOffset, endOffset := interfaces.OffsetNodeByNode(c.NameNode, c.Node)

			start := utils.GetPositionForOffset(f.Content, startOffset)
			end := utils.GetPositionForOffset(f.Content, endOffset)

			locations = append(locations, interfaces.Location{Uri: f.URI, Range: utils.Range{Start: start, End: end}})
		}
	}

	return locations
}

func handleDefinitionOfTag(locations []interfaces.Location, class *Class) []interfaces.Location {
	c := class.Snapshot()
	cOffset := c.Node.StartByte()
	file := c.File.Snapshot()
	cContent := file.Content

	start := utils.GetPositionForOffset(cContent, c.NameNode.StartByte()+cOffset)
	end := utils.GetPositionForOffset(cContent, c.NameNode.EndByte()+cOffset)

	return append(locations, interfaces.Location{Uri: file.URI, Range: utils.Range{Start: start, End: end}})
}
