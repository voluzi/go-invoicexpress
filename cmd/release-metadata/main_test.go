package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRejectsPositionalArguments(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"unexpected"}, &stdout, &stderr)

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if output := stderr.String(); !strings.Contains(output, "unexpected positional argument") || !strings.Contains(output, "Usage:") {
		t.Fatalf("stderr = %q, want error and usage", output)
	}
}
