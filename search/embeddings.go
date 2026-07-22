package search

import (
	"math"
	"sync"
	"ts_inspector/parser"
	"ts_inspector/utils"

	"github.com/hybridgroup/yzma/pkg/llama"
)

var model llama.Model
var context llama.Context

var mutex sync.Mutex

const EMBEDDING_BATCH_SIZE = 64
const EMBEDDING_BATCH_TOKENS = EMBEDDING_BATCH_SIZE * 128 // I'll always have less than 128 tokens in my embeddings

func GetEmbedding(text string) []float32 {
	mutex.Lock()

	vocab := llama.ModelGetVocab(model)
	tokens := llama.Tokenize(vocab, text, true, true)

	batch := llama.BatchGetOne(tokens)
	_, err := llama.Decode(context, batch)
	if err != nil {
		panic(err)
	}

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

	mutex.Unlock()

	return normalisedEmbedding
}

func GetEmbeddingBatch(texts []string) [][]float32 {
	mutex.Lock()

	vocab := llama.ModelGetVocab(model)
	allTokens := [][]llama.Token{}

	for _, text := range texts {
		tokens := llama.Tokenize(vocab, text, true, true)
		allTokens = append(allTokens, tokens)
	}

	mutex.Unlock()

	embeddings := [][]float32{}

	for i := 0; i < len(allTokens); i += EMBEDDING_BATCH_SIZE {
		end := i + EMBEDDING_BATCH_SIZE
		if end > len(allTokens) {
			end = len(allTokens)
		}

		embeddings = append(embeddings, GetEmbeddingsFromTokens(allTokens[i:end])...)
	}

	return embeddings
}

func GetEmbeddingsFromTokens(allTokens [][]llama.Token) [][]float32 {
	mutex.Lock()
	defer mutex.Unlock()

	mem, _ := llama.GetMemory(context)
	llama.MemoryClear(mem, true)

	ntokens := 0
	for _, tokens := range allTokens {
		ntokens += len(tokens)
	}

	batch := llama.BatchInit(int32(ntokens), 0, int32(len(allTokens)))
	defer llama.BatchFree(batch)

	for id, tokens := range allTokens {
		for i, token := range tokens {
			if len(tokens) == 0 {
				continue
			}

			isLastToken := i == len(tokens)-1
			batch.Add(token, llama.Pos(i), []llama.SeqId{llama.SeqId(id)}, isLastToken)
		}
	}

	_, err := llama.Decode(context, batch)
	if err != nil {
		panic(err)
	}

	embeddings := make([][]float32, len(allTokens))
	embeddingDimensions := llama.ModelNEmbd(model)

	for i := 0; i < len(allTokens); i++ {
		embedding, err := llama.GetEmbeddingsSeq(context, llama.SeqId(i), embeddingDimensions)
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

		for j, e := range embedding {
			normalisedEmbedding[j] = e * normal
		}

		embeddings[i] = normalisedEmbedding
	}

	return embeddings
}

func indexEmbeddings(interestingPoints []parser.InterestingPoint, ids []int64) error {
	indexEmbeddingsFaiss(interestingPoints, ids)
	err := indexEmbeddingsSqlite(interestingPoints, ids)
	if err != nil {
		return err
	}

	setEmbeddingsReady()

	return nil
}

func indexEmbeddingsFaiss(interestingPoints []parser.InterestingPoint, ids []int64) {
	if len(interestingPoints) == 0 || !utils.SemanticSearchEnableFaiss {
		return
	}

	vectors := make([]Vector, len(interestingPoints))

	for i, interestingPoint := range interestingPoints {
		ppText := preprocessText(interestingPoint.Text)

		vector := GetEmbedding(ppText)
		vectors[i] = Vector{ids[i], vector}
	}

	AddToFAISS(vectors)
}

func indexEmbeddingsSqlite(interestingPoints []parser.InterestingPoint, ids []int64) error {
	if len(interestingPoints) == 0 || !utils.SemanticSearchEnableSqlite {
		return nil
	}

	rows := make([]row, len(interestingPoints))
	texts := make([]string, len(interestingPoints))

	for i, interestingPoint := range interestingPoints {
		ppText := preprocessText(interestingPoint.Text)
		texts[i] = ppText

		rows[i] = row{id: ids[i], text: ppText}
	}

	embeddings := GetEmbeddingBatch(texts)
	for i, embedding := range embeddings {
		rows[i].embedding = embedding
	}

	return AddToSqlite(rows)
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

	llama.LogSet(llama.LogNormal)
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
	contextParams.NSeqMax = EMBEDDING_BATCH_SIZE
	contextParams.NBatch = EMBEDDING_BATCH_TOKENS
	contextParams.NUbatch = EMBEDDING_BATCH_TOKENS

	mContext, err := llama.InitFromModel(m, contextParams)
	if err != nil {
		panic(err)
	}

	model = m
	context = mContext
}
