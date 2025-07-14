package main

import "os"

func main() {
	command := os.Args[1]

	if command == "dataset" {
		dataset_gen()
	} else if command == "reward" {
		reward()
	}
}
