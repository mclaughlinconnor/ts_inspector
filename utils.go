package main

import (
	"log"
	"os"
)

func FileExists(filename string) bool {
  stat, err := os.Stat(filename)
  if err != nil {
    return false
  }
  return !stat.IsDir()

}

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
