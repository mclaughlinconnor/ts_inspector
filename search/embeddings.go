package search

import (
	"github.com/hybridgroup/yzma/pkg/llama"
)

var model llama.Model
var context llama.Context

func GetEmbedding(text string) []float32 {
	vocab := llama.ModelGetVocab(model)
	tokens := llama.Tokenize(vocab, text, true, true)

	batch := llama.BatchGetOne(tokens)
	llama.Decode(context, batch)

	embeddingDimensions := llama.ModelNEmbd(model)
	embedding, err := llama.GetEmbeddingsSeq(context, 0, embeddingDimensions)
	if err != nil {
		panic(err)
	}

	copyEmbedding := make([]float32, embeddingDimensions)
	copy(copyEmbedding, embedding)

	return copyEmbedding
}

func initEmbedding() {
	libPath, err := extractEmbeddedLibs()
	if err != nil {
		panic(err)
	}

	err = llama.Load(libPath)
	if err != nil {
		panic(err)
	}

	llama.LogSet(llama.LogSilent())
	llama.Init()

	modelPath, err := extractEmbeddedModel()
	if err != nil {
		panic(err)
	}

	m, err := llama.ModelLoadFromFile(modelPath, llama.ModelDefaultParams())
	if err != nil {
		panic(err)
	}

	contextParams := llama.ContextDefaultParams()
	contextParams.Embeddings = 1

	mContext, err := llama.InitFromModel(m, contextParams)
	if err != nil {
		panic(err)
	}

	model = m
	context = mContext
}
