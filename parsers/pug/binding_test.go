package pug_test

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/mclaughlinconnor/ts_inspector/parser/pug"
	"github.com/stretchr/testify/assert"
)

func TestGrammar(t *testing.T) {
	assert := assert.New(t)

	n, err := sitter.ParseCtx(context.Background(), []byte(`span Hello World`), pug.GetLanguage())
	assert.NoError(err)
	assert.Equal(
		`(source_file (tag (tag_name) (content (content))))`,
		n.String(),
	)
}
