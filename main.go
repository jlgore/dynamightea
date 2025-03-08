package main

import (
	"fmt"
	"os"

	"github.com/jg/dynamightea/cmd/dynamightea"
)

func main() {
	if err := dynamightea.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}