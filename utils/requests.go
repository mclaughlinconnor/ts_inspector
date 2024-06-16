package utils

import (
	"encoding/json"
	"io"
	"log"
	"ts_inspector/rpc"
)

var MostRecentId int

func TryParseRequest[T any](logger *log.Logger, contents []byte) T {
	var request T
	if err := json.Unmarshal(contents, &request); err != nil {
		logger.Printf("Could not parse: %s", err)
	}

	return request
}

func WriteResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))
}
