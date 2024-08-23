package parser

import (
	"bytes"
	"strings"
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
	"Inject":          true,
	"Injectable":      true,
	"Input":           true,
	"NgModule":        true,
	"Optional":        true,
	"Output":          true,
	"Pipe":            true,
	"Self":            true,
	"SkipSelf":        true,
	"ViewChild":       true,
	"ViewChildren":    true,
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

func FilenameFromUri(uri string) string {
	return strings.TrimPrefix(uri, `file://`)
}

func UriFromFilename(filename string) string {
	return `file://` + filename
}

func CStr2GoStr(b []byte) string {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		i = len(b)
	}

	return string(b[:i])
}
