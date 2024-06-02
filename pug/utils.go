package pug

import sitter "github.com/smacker/go-tree-sitter"

func IsVoidElement(tag_name string) bool {
	switch tag_name {
	case "area":
	case "base":
	case "br":
	case "col":
	case "embed":
	case "hr":
	case "img":
	case "input":
	case "link":
	case "meta":
	case "param":
	case "source":
	case "track":
	case "wbr":
		return true
	}

	return false
}

func pushRange(state *State, toPush string, nodeType *NodeType, pugRange *NodeRange) *State {
	if pugRange != nil {
		htmlLen := len(state.HtmlText)

		r := Range{
			HtmlStart: uint32(htmlLen),
			HtmlEnd:   uint32(htmlLen + len(toPush)),
			NodeType:  *nodeType,
			PugStart:  pugRange.StartIndex,
			PugEnd:    pugRange.EndIndex,
		}

		state.Ranges = append(state.Ranges, r)
	}

	state.HtmlText += toPush

	return state
}

func pushRangeSurround(state *State, toPush string, pugRange NodeRange, surround string, nodeType NodeType) {
	pushRange(state, surround, nil, nil)
	pushRange(state, toPush, &nodeType, &pugRange)
	pushRange(state, surround, nil, nil)
}

func getRange(node *sitter.Node) NodeRange {
	return NodeRange{
		StartIndex:    node.StartByte(),
		EndIndex:      node.EndByte(),
		StartPosition: node.StartPoint(),
		EndPosition:   node.StartPoint(),
	}
}

func offsetPreviousRange(state *State, offset int32) NodeRange {
	if len(state.Ranges) > 0 {
		lastRange := state.Ranges[len(state.Ranges)-1]
		return NodeRange{
			StartIndex: uint32(int32(lastRange.PugEnd) + offset),
			EndIndex:   uint32(int32(lastRange.PugEnd) + offset),
		}
	}

	return NodeRange{StartIndex: 0, EndIndex: 0}
}

func rangeAtPugLocation(charIndex uint32, state State) Range {
	for _, r := range state.Ranges {
		if r.PugStart <= charIndex && charIndex <= r.PugEnd {
			return r
		}
	}

	return Range{0, 0, 0, 0, EMPTY}
}

func rangeAtHtmlLocation(charIndex uint32, state State) Range {
	for _, r := range state.Ranges {
		if r.HtmlStart <= charIndex && charIndex <= r.HtmlEnd {
			return r
		}
	}

	return Range{0, 0, 0, 0, EMPTY}
}

func HtmlLocationToPugLocation(charIndex uint32, state State) uint32 {
	var closest *Range
	for _, r := range state.Ranges {
		if r.HtmlStart <= charIndex && charIndex <= r.HtmlEnd {
			return min(r.PugStart+(charIndex-r.HtmlStart), uint32(len(state.PugText)))
		}

		if closest == nil && r.HtmlEnd > charIndex {
			closest = &r
		}
	}

	return min(closest.PugStart+(charIndex-closest.HtmlStart), uint32(len(state.PugText)))
}

func PugLocationToHtmlLocation(charIndex uint32, state State) uint32 {
	var closest *Range
	for _, r := range state.Ranges {
		if r.PugStart <= charIndex && charIndex <= r.PugEnd {
			return min(r.HtmlStart+(charIndex-r.PugStart), uint32(len(state.HtmlText)))
		}

		if closest == nil && r.PugEnd > charIndex {
			closest = &r
		}
	}

	return min(closest.HtmlStart+(charIndex-closest.PugStart), uint32(len(state.HtmlText)))
}
