package main

import (
	"flag"
	"fmt"
	"os"

	invoicexpress "github.com/voluzi/go-invoicexpress"
	"github.com/voluzi/go-invoicexpress/internal/releasemetadata"
)

func main() {
	tag := flag.String("tag", "", "release tag to validate")
	changelogPath := flag.String("changelog", "CHANGELOG.md", "path to the changelog")
	flag.Parse()

	changelog, err := os.ReadFile(*changelogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read changelog: %v\n", err)
		os.Exit(1)
	}
	notes, err := releasemetadata.Validate(invoicexpress.Version, *tag, changelog)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(notes)
}
