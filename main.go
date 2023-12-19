package main

import (
	"log"

	cmd "github.com/chainguard-dev/gobump/cmd/gobump"
)

func main() {
	if err := cmd.RootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
