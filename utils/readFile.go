package utils

import (
	"log"
	"os"
)

func ReadFile(filename string) []byte {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	data := make([]byte, 10240)

	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}

	return data
}
