package utils

import (
	"os"
)

func ReadFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return make([]byte, 0), err
	}

	data := make([]byte, 10240)

	_, err = file.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
