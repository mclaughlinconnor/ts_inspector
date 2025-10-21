package utils

import (
	"os"
)

func ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return make([]byte, 0), err
	}

	return data, nil
}
