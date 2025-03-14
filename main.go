package main

import (
	"fmt"
	"os"

	"github.com/jlgore/dynamighTea/cmd/dynamightea"
)

func main() {
	if err := dynamightea.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}