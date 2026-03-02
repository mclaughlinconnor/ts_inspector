package search

import (
	"cmp"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"ts_inspector/parser"
	"ts_inspector/utils"
	"unicode"

	"github.com/fatih/camelcase"
)

var interestingPointsIdCache = make(map[int64]parser.InterestingPoint, 0)

type ClassResult struct {
	Class *parser.Class
	Score float32
}

type Result struct {
	Distance float32
	Id       int64
	Source   string // "embedding" or "fzf"
}

const (
	SortOrderEmbedding = 0
	SortOrderFzf       = 1
)

const (
	FzfResultsCount       = 10
	EmbeddingResultsCount = 40
)

func IndexState(state *parser.State) {
	interestingPoints := state.GetInterestingPoints()

	hash := fnv.New64a()
	hash.Sum64()

	ids := make([]int64, 0)
	for _, interestingPoint := range interestingPoints {
		ipId := interestingPoint.Id()

		hash.Write([]byte(ipId))
		id := int64(hash.Sum64())
		hash.Reset()

		interestingPointsIdCache[id] = interestingPoint
		ids = append(ids, id)
	}

	go indexEmbeddings(interestingPoints, ids)
	indexFzf(interestingPoints, ids)

	setSearchReady()
}

func InitSearch() {
	initEmbedding()
	initFAISS()
}

func FindInterestingPoints(text string) ([]parser.InterestingPoint, error) {
	if !canSearch() {
		return []parser.InterestingPoint{}, nil
	}

	ppText := preprocessText(text)

	results, err := SearchFAISS(ppText, EmbeddingResultsCount)
	if err != nil {
		return []parser.InterestingPoint{}, err
	}

	results = append(results, SearchFZF(ppText, FzfResultsCount)...)

	slices.SortFunc(results, func(a Result, b Result) int { return cmp.Compare(b.Distance, a.Distance) })
	results = makeUnique(results)

	interestingPoints := make([]parser.InterestingPoint, 0)
	for _, result := range results {
		ip, found := interestingPointsIdCache[result.Id]
		if !found {
			continue
		}

		if utils.Debug {
			ip.Text += fmt.Sprintf(" (%v) (%v)", result.Distance, result.Source)
		}

		interestingPoints = append(interestingPoints, ip)
	}

	return interestingPoints, nil
}

func makeUnique(results []Result) []Result {
	output := []Result{}
	seenIds := map[int64]bool{}

	for _, result := range results {
		seen, found := seenIds[result.Id]
		if found && seen {
			continue
		}

		seenIds[result.Id] = true
		output = append(output, result)
	}

	return output
}

func preprocessText(text string) string {
	partials := []string{}

	split := camelcase.Split(text)
	for _, s := range split {
		if s == " " {
			continue
		}

		ss := strings.FieldsFunc(s, unicode.IsPunct)
		partials = append(partials, ss...)
	}

	return strings.Join(partials, " ")
}
