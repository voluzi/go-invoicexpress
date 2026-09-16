package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	invoicexpress "github.com/voluzi/go-invoicexpress"
	"github.com/voluzi/go-invoicexpress/internal/releasemetadata"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("release-metadata", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "Usage: %s [options]\n", flags.Name())
		flags.PrintDefaults()
	}
	tag := flags.String("tag", "", "release tag to validate")
	changelogPath := flags.String("changelog", "CHANGELOG.md", "path to the changelog")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintf(stderr, "unexpected positional argument: %s\n", strings.Join(flags.Args(), " "))
		flags.Usage()
		return 2
	}

	changelog, err := os.ReadFile(*changelogPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read changelog: %v\n", err)
		return 1
	}
	notes, err := releasemetadata.Validate(invoicexpress.Version, *tag, changelog)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if _, err := fmt.Fprint(stdout, notes); err != nil {
		_, _ = fmt.Fprintf(stderr, "write release notes: %v\n", err)
		return 1
	}
	return 0
}
