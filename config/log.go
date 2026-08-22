package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func GetLogger(name string) (*log.Logger, error) {
	cacheDir, err := GetCacheDir("logs")
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	filename := name + "-" + timestamp + ".log"
	path := filepath.Join(cacheDir, filename)

	logfile, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("invalid file: %s, %w", filename, err)
	}

	return log.New(logfile, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile), nil
}
