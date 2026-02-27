package search

import (
	"math"

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

	normalisedEmbedding := make([]float32, embeddingDimensions)

	var sum float64
	for _, v := range embedding {
		sum += float64(v * v)
	}
	sum = math.Sqrt(sum)
	normal := float32(1.0 / sum)

	for i, e := range embedding {
		normalisedEmbedding[i] = e * normal
	}

	return normalisedEmbedding
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
