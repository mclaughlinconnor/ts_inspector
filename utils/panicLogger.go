package utils

import (
	"log"
	"runtime/debug"
)

func PanicLogger(logger *log.Logger) {
	if r := recover(); r != nil {
		logger.Println("Panicked with: ", r, "responding with empty response")
		logger.Println("Stack: ", string(debug.Stack()))
	}
}
