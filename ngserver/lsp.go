package ngserver

import (
	"bufio"
	"context"
	"log"
	"os/exec"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

var inputMessages = make(chan string)
var outputMessages = make(chan string)
var cancel context.CancelFunc

func Stop() {
	cancel()
}

var logger = utils.GetLogger("log-ngserver.txt")

func Start() {
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())

	exe, args := angularlsCmd()

	cmd := exec.CommandContext(ctx, exe, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		logger.Printf("cmd.Start() failed with '%s'\n", err)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Split(rpc.Split)
		for scanner.Scan() {
			msg := scanner.Bytes()
			method, contents, err := rpc.DecodeMessage(msg)
			logger.Printf("stdout '%s' '%s' '%s'", method, contents, err)
			HandleResponse(method, contents, msg)
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				logger.Printf("stderr: %s\n", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer stdin.Close()
		for {
			m := <-inputMessages
			logger.Printf("stdin: %s\n", m)
			_, err := stdin.Write([]byte(m))
			if err != nil {
				logger.Printf("stdin.Write() failed with '%s'\n", err)
			}
		}
	}()
}
