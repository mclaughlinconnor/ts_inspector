package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"ts_inspector/actions"
	"ts_inspector/analysis"
	traversetypescriptfiles "ts_inspector/ast/indexing"
	"ts_inspector/commands"
	"ts_inspector/lsp"
	"ts_inspector/parser"
	"ts_inspector/search"
	"ts_inspector/utils"
)

func main() {
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	utils.InitQueries()
	actions.InitActions()
	commands.InitCommands()
	analysis.InitAnalysers()

	args := flag.Args()
	if utils.LSP && len(args) == 0 {
		startLsp()
		return
	}

	if len(args) < 1 {
		return
	}

	logger := utils.GetLogger("indexer")

	state := parser.CreateState()
	filenames := traversetypescriptfiles.Index(args[0])
	for _, filename := range filenames {
		logger.Println(filename)
		err := parser.IndexFileFromIndexer(&state, filename)
		if err != nil {
			logger.Fatal(err)
		}
	}

	state.Postprocess()

	search.InitSearch()
	search.IndexState(&state)

	logger.Println("Done")
}

func startLsp() {
	go lsp.Start()

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool, 1)

	shutdown := func() {
		done <- true
	}

	go func() {
		select {
		case <-sigs:
			shutdown()
		case <-lsp.Shutdown:
			shutdown()
		}
	}()

	<-done
	fmt.Println("Exiting")
}
