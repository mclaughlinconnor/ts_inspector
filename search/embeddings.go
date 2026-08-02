package search

import (
	"math"
	"strings"
	"sync"
	"ts_inspector/config"
	"ts_inspector/parser"
	"unsafe"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
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
	embeddings := make([][]float32, len(texts))

	precalculatedEmbeddings, err := GetEmbeddingsFromSqlite(texts)
	if err != nil {
		precalculatedEmbeddings = map[string][]float32{}
	}

	for i, text := range texts {
		embedding, found := precalculatedEmbeddings[text]
		if !found {
			continue
		}

		embeddings[i] = embedding
	}

	mutex.Lock()

	vocab := llama.ModelGetVocab(model)
	allTokens := [][]llama.Token{}

	for i, text := range texts {
		if embeddings[i] != nil {
			continue
		}

		tokens := llama.Tokenize(vocab, text, true, true)
		allTokens = append(allTokens, tokens)
	}

	mutex.Unlock()

	embeddingIndex := 0

	for i := 0; i < len(allTokens); i += EMBEDDING_BATCH_SIZE {
		end := i + EMBEDDING_BATCH_SIZE
		if end > len(allTokens) {
			end = len(allTokens)
		}

		for embeddings[embeddingIndex] != nil {
			embeddingIndex++
		}

		for _, calculatedEmbedding := range GetEmbeddingsFromTokens(allTokens[i:end]) {
			for embeddings[embeddingIndex] != nil {
				embeddingIndex++
			}

			embeddings[embeddingIndex] = calculatedEmbedding
		}
	}

	return embeddings
}

func GetEmbeddingsFromSqlite(texts []string) (map[string][]float32, error) {
	embeddings := map[string][]float32{}

	prefix := "SELECT text, embedding FROM vec_cache WHERE "

	sb := strings.Builder{}
	sb.WriteString(prefix)

	args := []any{}
	for i, text := range texts {
		if i < len(texts)-1 && (i%500 != 0 || i == 0) {
			if len(args) > 0 {
				sb.WriteString(" OR ")
			}

			sb.WriteString("text = ?")
			args = append(args, text)

			continue
		}

		sb.WriteString(";")

		rows, err := db.Query(sb.String(), args...)
		if err != nil {
			return map[string][]float32{}, err
		}

		sb.Reset()
		sb.WriteString(prefix)

		for rows.Next() {
			var text string
			var embedding []byte

			if err := rows.Scan(&text, &embedding); err != nil {
				return nil, err
			}

			e := unsafe.Slice((*float32)(unsafe.Pointer(&embedding[0])), len(embedding)/4)
			embeddings[text] = e
		}

		args = []any{}
	}

	return embeddings, nil
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

func indexEmbeddings(interestingPoints []parser.InterestingPoint, ids []int64, rootPath string) error {
	indexEmbeddingsFaiss(interestingPoints, ids)
	err := indexEmbeddingsSqlite(interestingPoints, ids, rootPath)
	if err != nil {
		return err
	}

	setEmbeddingsReady()

	return nil
}

func indexEmbeddingsFaiss(interestingPoints []parser.InterestingPoint, ids []int64) {
	if len(interestingPoints) == 0 || !config.SemanticSearchEnableFaiss {
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

func indexEmbeddingsSqlite(interestingPoints []parser.InterestingPoint, ids []int64, rootPath string) error {
	if len(interestingPoints) == 0 || !config.SemanticSearchEnableSqlite {
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

	err := AddToSqlite("vec_cache", rows, []string{"embedding", "text"}, []string{"text"}, func(args []any, r row) []any {
		vector := unsafe.Slice((*byte)(unsafe.Pointer(&r.embedding[0])), len(r.embedding)*4)

		return append(args, vector, r.text)
	})

	if err != nil {
		return err
	}

	DeleteInterestingFromUri(rootPath)

	return AddToSqlite("vec_interesting_points", rows, []string{"id", "embedding", "text", "rootPath"}, []string{}, func(args []any, r row) []any {
		vector, err := sqlite_vec.SerializeFloat32(r.embedding)
		if err != nil {
			return args
		}

		return append(args, r.id, vector, r.text, rootPath)
	})
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
