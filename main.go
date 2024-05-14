package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ts_inspector/analysis"
	"ts_inspector/lsp"
	"ts_inspector/parser"
)

func main() {
	if len(os.Args) < 2 {
		lsp.Start()
	}

	filename, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	parser.InitQueries()
	state := parser.State{}
	state, err = parser.HandleFile(state, `file://`+filename, "typescript", 0, log.New(os.Stdout, "[ts_inspector]", log.Ldate|log.Ltime|log.Lshortfile))

	if err != nil {
		log.Fatal(err)
	}

	j, err := json.MarshalIndent(state, "", "  ")
	fmt.Println(string(j))

	analyses := analysis.Analyse(state[filename])

	j, err = json.MarshalIndent(analyses, "", "  ")
	fmt.Println(string(j))
}
