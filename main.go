package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"ts_inspector/actions"
	traversetypescriptfiles "ts_inspector/ast/indexing"
	"ts_inspector/commands"
	"ts_inspector/lsp"
	"ts_inspector/ngserver"
	"ts_inspector/parser"
	"ts_inspector/utils"
)

func main() {
	if len(os.Args) == 0 {
		startLsp()
	}

	logger := utils.GetLogger("indexing")

	utils.InitQueries()
	actions.InitActions()
	commands.InitCommands()

	files := traversetypescriptfiles.Index(os.Args[1])

	state := parser.State{Files: map[string]parser.File{}, RootURI: os.Args[1]}
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
	go ngserver.Start()
	go lsp.Start()

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool, 1)

	shutdown := func() {
		ngserver.Stop()
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
