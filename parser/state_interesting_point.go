package parser

import (
	"math/rand"
	"strconv"
	"ts_inspector/interfaces"
	"ts_inspector/utils"
)

type InterestingPoint struct {
	Kind interfaces.TSymbolKind
	Text string

	endOffset   uint32
	fileContent string
	location    *interfaces.Location
	startOffset uint32
	uri         string

	randint int
}

func (i *InterestingPoint) Id() string {
	if i.randint == 0 {
		i.randint = rand.Int()
	}

	return strconv.FormatUint(uint64(i.startOffset), 10) + strconv.FormatUint(uint64(i.endOffset), 10) + i.Text + strconv.FormatUint(uint64(i.Kind), 10) + strconv.FormatUint(uint64(i.randint), 10)
}

func (i *InterestingPoint) ResolveLocation() interfaces.Location {
	if i.location != nil {
		return *i.location
	}

	startPosition := utils.GetPositionForOffset(i.fileContent, i.startOffset)
	endPosition := utils.GetPositionForOffset(i.fileContent, i.endOffset)

	location := interfaces.Location{Uri: i.uri, Range: utils.Range{End: endPosition, Start: startPosition}}

	i.location = &location

	return location
}

func (i *InterestingPoint) SetPosition(startOffset uint32, endOffset uint32) {
	i.endOffset = endOffset
	i.startOffset = startOffset
}

func (i *InterestingPoint) SetFile(content string, uri string) {
	i.fileContent = content
	i.uri = uri
}
