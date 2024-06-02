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

	go func() {
		<-sigs // block until SIGINT or SIGTERM
		ngserver.Stop()
		done <- true
	}()

	<-done // block until done
	fmt.Println("Exiting")
}
