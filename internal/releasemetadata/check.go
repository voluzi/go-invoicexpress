package releasemetadata

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	stableSemver   = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	sectionHeading = regexp.MustCompile(`(?m)^##[\t ]+[^\r\n]+[\t ]*\r?$`)
)

// Validate verifies release metadata and returns only the matching changelog
// section body for use as GitHub Release notes.
func Validate(version, tag string, changelog []byte) (string, error) {
	if !stableSemver.MatchString(version) {
		return "", fmt.Errorf("version %q is not a stable semantic version", version)
	}
	if tag != "v"+version {
		return "", fmt.Errorf("tag %q does not match version %q", tag, version)
	}

	heading := regexp.MustCompile(`(?m)^## \[` + regexp.QuoteMeta(version) + `\] - ([0-9]{4}-[0-9]{2}-[0-9]{2})[\t ]*\r?$`)
	matches := heading.FindAllSubmatchIndex(changelog, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("changelog must contain exactly one dated section for %s, found %d", version, len(matches))
	}
	match := matches[0]
	date := string(changelog[match[2]:match[3]])
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", fmt.Errorf("changelog section for %s has invalid date %q: %w", version, date, err)
	}

	contentEnd := len(changelog)
	if next := sectionHeading.FindIndex(changelog[match[1]:]); next != nil {
		contentEnd = match[1] + next[0]
	}
	notes := strings.TrimSpace(string(changelog[match[1]:contentEnd]))
	if notes == "" {
		return "", fmt.Errorf("changelog section for %s is empty", version)
	}
	return notes + "\n", nil
}
