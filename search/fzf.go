package search

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

var fzfIndex []string
var fzfLabels []int64

const (
	slab16Size int = 100 * 1024 // 200KB * 32 = 12.8MB
	slab32Size int = 2048       // 8KB * 32 = 256KB
)

var slab = util.MakeSlab(slab16Size, slab32Size)

func AddToFZF(text string, labels int64) {
	fzfIndex = append(fzfIndex, strings.ToLower(text))
	fzfLabels = append(fzfLabels, labels)
}

func SearchFZF(queryText string, resultsCount int64) []Result {
	query := []rune(strings.ToLower(queryText))

	results := make([]Result, len(fzfIndex))
	var sum float64
	for i, text := range fzfIndex {
		chars := util.ToChars([]byte(text))

		result, _ := algo.FuzzyMatchV2(false, true, false, &chars, query, true, slab)
		sum += float64(result.Score * result.Score)

		results[i] = Result{Distance: float32(result.Score), Id: int64(fzfLabels[i]), Source: "fzf"}
	}

	sum = math.Sqrt(sum)
	normal := float32(1.0 / sum)

	finalResults := make([]Result, 0)
	for _, result := range results {
		if result.Distance <= 1 {
			continue
		}

		result.Distance = (result.Distance * normal) + SortOrderFzf
		finalResults = append(finalResults, result)
	}

	slices.SortFunc(finalResults, func(a Result, b Result) int { return cmp.Compare(b.Distance, a.Distance) })

	return finalResults[:min(resultsCount, int64(len(finalResults)))]
}
