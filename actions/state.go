package actions

import "ts_inspector/utils"

type Edit struct {
	Range utils.Range

	Text string
}

type Edits = []Edit
