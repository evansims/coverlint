package main

import (
	"os"

	"github.com/evansims/coverlint/internal/coverage"
)

func main() {
	if err := coverage.Run(); err != nil {
		coverage.EmitAnnotation("error", err.Error())
		os.Exit(1)
	}
}
