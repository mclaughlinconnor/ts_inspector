package typescript_test

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/mclaughlinconnor/ts_inspector/parser/typescript"
	"github.com/stretchr/testify/assert"
)

func TestGrammar(t *testing.T) {
	assert := assert.New(t)

	n, err := sitter.ParseCtx(context.Background(), []byte(`console.log("Hello World")`), typescript.GetLanguage())
	assert.NoError(err)
	assert.Equal(
		`(source_file (tag (tag_name) (content (content))))`,
		n.String(),
	)
}
