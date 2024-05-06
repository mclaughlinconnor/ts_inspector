package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"ts_inspector/rpc"
)

func Start() {
	logger := getLogger("/home/connor/Development/ts_inspector/log.txt")
	logger.Println("Started")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()
		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			logger.Printf("Error: %s", err)
			continue
		}

		handleMessage(logger, writer, method, contents)
	}
}

func handleMessage(logger *log.Logger, writer io.Writer, method string, contents []byte) {
	logger.Printf("Received msg with method: %s", method)
	var request InitializeRequest
	if err := json.Unmarshal(contents, &request); err != nil {
		logger.Printf("Hey, we couldn't parse this: %s", err)
	}

	switch method {
	case "initialize":
		msg := NewInitializeResponse(request.ID)
		writeResponse(writer, msg)
	}

	logger.Print("Sent the reply")
}

func writeResponse(writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)
	writer.Write([]byte(reply))

}

func getLogger(filename string) *log.Logger {
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Invalid file: %s", filename))
	}

	return log.New(logfile, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile)
}
