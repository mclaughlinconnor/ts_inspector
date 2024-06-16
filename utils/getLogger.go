package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

func GetLogger(f string) *log.Logger {
	timestamp := (time.Now().UTC().Format(time.RFC3339))

	filename := "/home/connor/Development/ts_inspector/logs/" + f + "-" + timestamp + ".log"

	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Invalid file: %s", filename))
	}

	return log.New(logfile, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile)
}
