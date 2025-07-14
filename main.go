package main

import "os"

func main() {
	if len(os.Args) <= 1 {
		println("Must have command: 'dataset' or 'reward'")
		os.Exit(69)
	}

	command := os.Args[1]

	if command == "dataset" {
		dataset_gen()
	} else if command == "reward" {
		reward()
	} else {
		println("'dataset' or 'reward'")
	}
}
