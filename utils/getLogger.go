package utils

import (
	"fmt"
	"log"
	"os"
	"time"
	"ts_inspector/config"
)

func GetLogger(f string) *log.Logger {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	filename := config.LogsPath + f + "-" + timestamp + ".log"

	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Invalid file: %s", filename))
	}

	return log.New(logfile, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile)
}
