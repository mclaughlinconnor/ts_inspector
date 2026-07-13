package utils

import (
	"log"
	"time"
)

func Timer(logger *log.Logger, message string, start time.Time, enabled bool) {
	if !enabled {
		return
	}

	logger.Printf("%v: %v\n", message, time.Since(start))
}
