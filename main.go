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

	logger := utils.GetLogger("indexing")

	projectRoot := "../angular-tour-of-heroes"
	filenames := traversetypescriptfiles.Index(projectRoot)
	state := parser.CreateState(projectRoot)

	var err error
	for _, filename := range filenames {
		err = parser.IndexFileFromIndexer(&state, filename) // todo: are these filenames absolute?
		if err != nil {
			logger.Fatal(err)
		}
	}

	state.Postprocess()

	if len(os.Args) == 1 {
		startLsp(&state)
		return
	}
}

func startLsp(state *parser.State) {
	go lsp.Start(state)

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
