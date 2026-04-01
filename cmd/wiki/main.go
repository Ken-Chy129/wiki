package main

import (
	"fmt"
	"os"

	"github.com/Ken-Chy129/wiki/internal/wiki"
)

func main() {
	if err := wiki.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
