package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"ts_inspector/lsp"
	"ts_inspector/ngserver"
)

func main() {
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
