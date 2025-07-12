package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

func GetLogger(f string) *log.Logger {
	timestamp := (time.Now().UTC().Format(time.RFC3339))

	filename := "/home/connor/Development/dataset_gen/logs/" + f + "-" + timestamp + ".log"

	logfile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Invalid file: %s", filename))
	}

	return log.New(logfile, "[dataset_gen]", log.Ldate|log.Ltime|log.Lshortfile)
}
