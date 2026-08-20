package utils

import (
	"encoding/json"
	"io"
	"log"
	"ts_inspector/rpc"
)

func TryParseRequest[T any](logger *log.Logger, contents []byte) T {
	var request T
	if err := json.Unmarshal(contents, &request); err != nil {
		logger.Printf("Could not parse: %s\n%s", err, contents)
	}

	return request
}

func WriteResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	_, e := writer.Write([]byte(reply))
	if e != nil {
		panic(e)
	}
}
