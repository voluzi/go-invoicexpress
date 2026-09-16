package releasemetadata

import "testing"

func TestValidateExtractsOnlyMatchingDatedSection(t *testing.T) {
	changelog := []byte(`# Changelog

## [Unreleased]

- Future work.

## [0.3.0] - 2026-09-16

### Added
- Release feature.

## Maintenance note

- Not part of the release section.

## [0.2.2] - 2026-09-16

- Older fix.
`)
	notes, err := Validate("0.3.0", "v0.3.0", changelog)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	want := "### Added\n- Release feature.\n"
	if notes != want {
		t.Fatalf("notes = %q, want %q", notes, want)
	}
}

func TestValidatePreservesReleaseNoteWhitespace(t *testing.T) {
	changelog := []byte("## [0.3.0] - 2026-09-16\n\n    indented example\n- Markdown hard break  \n\n## [0.2.2] - 2026-09-16\n\n- Older.\n")

	notes, err := Validate("0.3.0", "v0.3.0", changelog)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	want := "    indented example\n- Markdown hard break  \n"
	if notes != want {
		t.Fatalf("notes = %q, want %q", notes, want)
	}
}

func TestValidateIgnoresSectionHeadingsInsideFencedCode(t *testing.T) {
	changelog := []byte("## [0.3.0] - 2026-09-16\n\n### Example\n```markdown\n## Example heading\n```\n- Still release notes.\n\n## Maintenance note\n\n- Not release notes.\n")

	notes, err := Validate("0.3.0", "v0.3.0", changelog)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	want := "### Example\n```markdown\n## Example heading\n```\n- Still release notes.\n"
	if notes != want {
		t.Fatalf("notes = %q, want %q", notes, want)
	}
}

func TestValidateDoesNotTreatStandaloneInlineCodeAsFence(t *testing.T) {
	changelog := []byte("## [0.3.0] - 2026-09-16\n\n```inline code```\n- Current notes.\n\n## [0.2.2] - 2026-09-15\n\n- Older notes.\n")

	notes, err := Validate("0.3.0", "v0.3.0", changelog)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	want := "```inline code```\n- Current notes.\n"
	if notes != want {
		t.Fatalf("notes = %q, want %q", notes, want)
	}
}

func TestValidateRejectsMetadataDrift(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		tag       string
		changelog string
	}{
		{name: "prerelease version", version: "0.3.0-rc.1", tag: "v0.3.0-rc.1", changelog: "## [0.3.0-rc.1] - 2026-09-16\n\n- Notes.\n"},
		{name: "build metadata", version: "0.3.0+build", tag: "v0.3.0+build", changelog: "## [0.3.0+build] - 2026-09-16\n\n- Notes.\n"},
		{name: "tag mismatch", version: "0.3.0", tag: "v0.3.1", changelog: "## [0.3.0] - 2026-09-16\n\n- Notes.\n"},
		{name: "missing section", version: "0.3.0", tag: "v0.3.0", changelog: "## [0.2.2] - 2026-09-16\n\n- Notes.\n"},
		{name: "duplicate section", version: "0.3.0", tag: "v0.3.0", changelog: "## [0.3.0] - 2026-09-16\n\n- One.\n\n## [0.3.0] - 2026-09-17\n\n- Two.\n"},
		{name: "undated section", version: "0.3.0", tag: "v0.3.0", changelog: "## [0.3.0]\n\n- Notes.\n"},
		{name: "invalid date", version: "0.3.0", tag: "v0.3.0", changelog: "## [0.3.0] - 2026-02-30\n\n- Notes.\n"},
		{name: "empty section", version: "0.3.0", tag: "v0.3.0", changelog: "## [0.3.0] - 2026-09-16\n\n## [0.2.2] - 2026-09-16\n\n- Older.\n"},
		{name: "whitespace-only section", version: "0.3.0", tag: "v0.3.0", changelog: "## [0.3.0] - 2026-09-16\n\n  \t  \n\n## [0.2.2] - 2026-09-16\n\n- Older.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if notes, err := Validate(tt.version, tt.tag, []byte(tt.changelog)); err == nil {
				t.Fatalf("notes = %q, want an error", notes)
			}
		})
	}
}
