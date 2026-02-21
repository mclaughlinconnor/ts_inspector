package search

import (
	"hash/fnv"
	"ts_inspector/parser"
)

var classIdCache = make(map[int64]*parser.Class, 0)

type ClassResult struct {
	Class *parser.Class
	Score float32
}

func IndexState(state *parser.State) {
	classes := *state.GetClasses()

	hash := fnv.New64a()
	hash.Sum64()

	vectors := make([]Vector, len(classes))
	i := 0
	for _, class := range classes {
		classId := class.Id()

		hash.Write([]byte(classId))
		id := int64(hash.Sum64())
		hash.Reset()

		classIdCache[id] = class

		vector := GetEmbedding(class.Snapshot().Name)

		vectors[i] = Vector{id, vector}
		i++
	}

	AddToFAISS(vectors)
}

func InitSearch() {
	initEmbedding()
	initFAISS()
}

func FindClass(text string) ([]ClassResult, error) {
	query := GetEmbedding(text)
	results, err := SearchFAISS(query, 5)
	if err != nil {
		return []ClassResult{}, err
	}

	classes := make([]ClassResult, 0)
	for _, result := range results {
		c, found := classIdCache[result.Id]
		if !found {
			continue
		}

		classes = append(classes, ClassResult{Class: c, Score: result.Distance})
	}

	return classes, nil
}
