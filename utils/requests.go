package utils

import (
	"encoding/json"
	"io"
	"log"
	"sync"
	"ts_inspector/rpc"
)

func TryParseRequest[T any](logger *log.Logger, contents []byte) T {
	var request T
	if err := json.Unmarshal(contents, &request); err != nil {
		logger.Printf("Could not parse: %s\n%s", err, contents)
	}

	return request
}

type Writer struct {
	sync.Mutex
	writer io.Writer
}

func NewWriter(writer io.Writer) *Writer {
	return &Writer{sync.Mutex{}, writer}
}

func WriteResponse(writer *Writer, msg any) {
	reply := rpc.EncodeMessage(msg)

	writer.Lock()
	defer writer.Unlock()

	_, e := writer.writer.Write([]byte(reply))
	if e != nil {
		panic(e)
	}
}
