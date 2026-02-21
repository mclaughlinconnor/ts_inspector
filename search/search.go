package search

import (
	"hash/fnv"
	"ts_inspector/parser"
)

var interestingPointsIdCache = make(map[int64]parser.InterestingPoint, 0)

type ClassResult struct {
	Class *parser.Class
	Score float32
}

func IndexState(state *parser.State) {
	interestingPoints := state.GetInterestingPoints()

	hash := fnv.New64a()
	hash.Sum64()

	vectors := make([]Vector, len(interestingPoints))
	i := 0
	for _, interestingPoint := range interestingPoints {
		ipId := interestingPoint.Id()

		hash.Write([]byte(ipId))
		id := int64(hash.Sum64())
		hash.Reset()

		interestingPointsIdCache[id] = interestingPoint

		vector := GetEmbedding(interestingPoint.Text)

		vectors[i] = Vector{id, vector}
		i++
	}

	if len(vectors) == 0 {
		return
	}

	AddToFAISS(vectors)
}

func InitSearch() {
	initEmbedding()
	initFAISS()
}

func FindInterestingPoints(text string, resultsCount int64) ([]parser.InterestingPoint, error) {
	query := GetEmbedding(text)
	results, err := SearchFAISS(query, resultsCount)
	if err != nil {
		return []parser.InterestingPoint{}, err
	}

	interestingPoints := make([]parser.InterestingPoint, 0)
	for _, result := range results {
		ip, found := interestingPointsIdCache[result.Id]
		if !found {
			continue
		}

		interestingPoints = append(interestingPoints, ip)
	}

	return interestingPoints, nil
}
