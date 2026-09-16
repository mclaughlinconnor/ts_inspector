package commands

import (
	"ts_inspector/interfaces"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func GetReviewFindings(writer *utils.Writer, state *parser.State, args *any) (map[string]utils.TextEdits, error) {
	locations := []interfaces.Location{}

	for _, finding := range state.GetReviewFindings() {
		uri := parser.UriFromFilename(finding.Metadata.FilePath)

		file, found := state.GetFile(finding.Metadata.FilePath)
		if !found {
			locations = append(locations, interfaces.Location{Uri: uri, Range: utils.ZeroRange()})
			continue
		}

		line := finding.Metadata.NewLine - 1
		start := file.GetOffsetForLineNumber(line)
		end := file.GetOffsetForLineNumber(line+1) - 1

		r := file.GetLocationForOffset(int(start), int(end)).Range

		locations = append(locations, interfaces.Location{Uri: uri, Range: r})
	}

	response := interfaces.ShowLocationsNotification{
		RPC: "2.0", Method: "ts_inspector/showLocations",
		Params: locations,
	}

	utils.WriteResponse(writer, response)

	return map[string]utils.TextEdits{}, nil
}
