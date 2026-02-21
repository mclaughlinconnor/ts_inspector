package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
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

	if utils.LSP && len(os.Args) == 1 {
		startLsp()
		return
	}

	if len(os.Args) != 2 {
		return
	}

	logger := utils.GetLogger("indexer")

	state := parser.CreateState()
	filenames := traversetypescriptfiles.Index(os.Args[1])
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

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter text: ")
		t, _ := reader.ReadString('\n')

		text := strings.TrimSpace(t)
		if text == "exit" {
			break
		}

		classes, err := search.FindClass(text)
		if err != nil {
			panic(err)
		}

		for _, c := range classes {
			fmt.Printf("  %v -- %v\n", c.Class.Snapshot().Name, c.Score)
		}
	}

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
