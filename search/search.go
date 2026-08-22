package search

import (
	"cmp"
	"fmt"
	"hash/fnv"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
	"ts_inspector/config"
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
	Source   string // "faiss" or "fzf" or "sqlite"
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

	ips := make([]parser.InterestingPoint, 0)
	ids := make([]int64, 0)

	for _, interestingPoint := range interestingPoints {
		ipId := interestingPoint.Id()

		hash.Write([]byte(ipId))
		id := int64(hash.Sum64())
		hash.Reset()

		_, found := interestingPointsIdCache[id]
		if !found {
			interestingPointsIdCache[id] = interestingPoint
			ids = append(ids, id)
			ips = append(ips, interestingPoint)
		}
	}

	go func() {
		err := indexEmbeddings(ips, ids, state.GetRootPath())
		if err != nil {
			state.Logger.Println(err)
		}
	}()
	indexFzf(ips, ids)

	setSearchReady()
}

func InitSearch() {
	initEmbedding()
	initFAISS()

	err := initSqlite()
	if err != nil {
		panic(err)
	}
}

func FindInterestingPoints(logger *log.Logger, text string) ([]parser.InterestingPoint, error) {
	defer utils.Timer(logger, "FindInterestingPoints", time.Now(), config.Debug)
	if !canSearch() {
		return []parser.InterestingPoint{}, nil
	}

	ppText := preprocessText(text)

	wg := sync.WaitGroup{}

	var faissResults []Result
	var fzfResults []Result
	var sqliteResults []Result

	var err error

	wg.Go(func() {
		defer utils.Timer(logger, "SearchFaiss", time.Now(), config.Debug)
		results, e := SearchFAISS(ppText, EmbeddingResultsCount)
		if e != nil {
			err = e
			return
		}

		faissResults = results
	})

	wg.Go(func() {
		defer utils.Timer(logger, "SearchFZF", time.Now(), config.Debug)
		results := SearchFZF(ppText, FzfResultsCount)
		fzfResults = results
	})

	wg.Go(func() {
		defer utils.Timer(logger, "sqliteSearch", time.Now(), config.Debug)
		results, e := SearchSqlite(ppText, EmbeddingResultsCount)
		if e != nil {
			err = e
			return
		}

		sqliteResults = results
	})

	wg.Wait()

	if err != nil {
		return []parser.InterestingPoint{}, err
	}

	results := []Result{}
	results = append(results, faissResults...)
	results = append(results, fzfResults...)
	results = append(results, sqliteResults...)

	slices.SortFunc(results, func(a Result, b Result) int { return cmp.Compare(b.Distance, a.Distance) })
	results = makeUnique(results)

	interestingPoints := make([]parser.InterestingPoint, 0)
	for _, result := range results {
		ip, found := interestingPointsIdCache[result.Id]
		if !found {
			continue
		}

		if config.Debug {
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
