package main

import (
	"bufio"
	"dataset_gen/utils"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
)

func reward() {
	utils.InitQueries()

	writer := bufio.NewWriter(os.Stdout)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	// Try os.Args

	_, _ = utils.ParseFile(false, text, utils.Pug, nil,
		func(root *sitter.Node, content []byte, state any) (any, error) {
			if root.HasError() {
				writer.WriteString("-1")
				return nil, nil
			}

			writer.WriteString("1")
			return nil, nil
		})

	writer.Flush()
}
