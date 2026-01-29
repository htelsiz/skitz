package main

import (
	"fmt"
	"os"

	"github.com/htelsiz/skitz/internal/app"
)

func main() {
	resource := ""
	if len(os.Args) > 1 {
		resource = os.Args[1]
	}

	if err := app.Run(resource); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
