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
	traversetypescriptfiles "ts_inspector/ast/indexing"
	"ts_inspector/commands"
	"ts_inspector/lsp"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func main() {
	if len(os.Args) == 1 {
		startLsp()
		return
	}

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

	logger := utils.GetLogger("indexing")

	file := "../angular-tour-of-heroes"
	files := traversetypescriptfiles.Index(file)
	state := parser.State{Files: map[string]parser.File{}, RootURI: file}

	var err error
	for _, file := range files {
		state, err = parser.HandleFile(state, file, "", 0, "", logger)
		if err != nil {
			logger.Fatal(err)
		}
	}

	fmt.Print(state)
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
