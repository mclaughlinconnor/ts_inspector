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

func FindDefinition(file *File, cursorOffset uint32) []interfaces.Location {
	locations := make([]interfaces.Location, 0)

	tagName, found := ast.GetTagNameAtOffset(file.Snapshot().Content, cursorOffset)
	if found {
		for _, c := range file.Snapshot().Classes {
			if !c.HasComponent() {
				continue
			}

			components := c.Angular.Component.GetAvailableComponents()
			for _, c := range components {
				cContent := c.File.Snapshot().Content

				if c.Angular.Component.Selector == tagName {
					start := GetPositionForOffset(cContent, c.NameNode.StartByte()+c.Node.StartByte())
					end := GetPositionForOffset(cContent, c.NameNode.EndByte()+c.Node.StartByte())

					locations = append(locations, interfaces.Location{Uri: c.File.Snapshot().URI, Range: utils.Range{Start: start, End: end}})
				}
			}
		}
	}

	return locations
}
