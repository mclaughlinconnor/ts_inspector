package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"ts_inspector/parser"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("A filename must be provided")
	}

	filename, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	parser.InitQueries()
	usages, _ := parser.HandleTypeScriptFile(filename)

	j, err := json.MarshalIndent(usages, "", "  ")

	fmt.Println(string(j))
}
