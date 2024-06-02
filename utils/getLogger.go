package utils

import (
	"fmt"
	"log"
	"os"
)

func GetLogger(filename string) *log.Logger {
	filename = "/home/connor/Development/ts_inspector/" + filename
	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Invalid file: %s", filename))
	}

	return log.New(logfile, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile)
}
