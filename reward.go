package main

import (
	"bufio"
	"dataset_gen/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"os"
	"strings"
)

func reward() {
	if model == "mellum" {
		filename_token = "<filename>"
		fim_suffix_token = "<fim_suffix>"
		fim_prefix_token = "<fim_prefix>"
		fim_middle_token = "<fim_middle>"
	} else if model == "qwen" {
		filename_token = "<|file_sep|>"
		fim_suffix_token = "<|fim_suffix|>"
		fim_prefix_token = "<|fim_prefix|>"
		fim_middle_token = "<|fim_middle|>"
	}

	utils.InitQueries()

	// Try os.Args
	writer := bufio.NewWriter(os.Stdout)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	pug := ""

	files := strings.Split(text, filename_token)
	for _, file := range files {
		lines := strings.SplitN(file, "\n", 2)

		if len(lines) < 2 {
			continue
		}

		firstLine := lines[0]

		if !strings.HasSuffix(firstLine, ".pug") {
			continue
		}

		pug = lines[1]
	}

	parts := strings.SplitN(pug, fim_prefix_token, 2)
	_ = parts[0]
	a := parts[1]

	parts = strings.SplitN(a, fim_suffix_token, 2)
	prefix := parts[0]
	b := parts[1]

	parts = strings.SplitN(b, fim_middle_token, 2)
	suffix := parts[0]
	middle := parts[1]

	final_pug_document := prefix + middle + suffix
	println(final_pug_document)

	is_good := false

	attempt_parse := func() bool {
		_, _ = utils.ParseFile(false, final_pug_document, utils.Pug, nil,
			func(root *sitter.Node, content []byte, state any) (any, error) {
				if root.HasError() {
					return nil, nil
				}

				is_good = true
				return nil, nil
			})

		if is_good {
			return true
		}

		if strings.HasSuffix(strings.TrimSpace(middle), ",") {
			final_pug_document = prefix + middle + ")" + suffix
			println(final_pug_document)
		}

		_, _ = utils.ParseFile(false, final_pug_document, utils.Pug, nil,
			func(root *sitter.Node, content []byte, state any) (any, error) {
				if root.HasError() {
					return nil, nil
				}

				is_good = true
				return nil, nil
			})

		if is_good {
			return true
		}

		return false
	}

	// If middle ends with a comma, try adding a closing ). Maybe it's a `span(\nattr='f',\n`

	if attempt_parse() {
		writer.WriteString("1")
	} else {
		writer.WriteString("-1")
	}

	writer.Flush()
}
