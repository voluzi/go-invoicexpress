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

	section := string(changelog[match[1]:])
	contentEnd := len(section)
	if next := nextSectionHeading(section); next >= 0 {
		contentEnd = next
	}
	notes := strings.Trim(section[:contentEnd], "\r\n")
	if strings.TrimSpace(notes) == "" {
		return "", fmt.Errorf("changelog section for %s is empty", version)
	}
	return notes + "\n", nil
}

func nextSectionHeading(section string) int {
	var fence byte
	var fenceLength int
	for offset := 0; offset < len(section); {
		lineEnd := strings.IndexByte(section[offset:], '\n')
		nextOffset := len(section)
		if lineEnd >= 0 {
			lineEnd += offset
			nextOffset = lineEnd + 1
		} else {
			lineEnd = len(section)
		}
		line := strings.TrimSuffix(section[offset:lineEnd], "\r")
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 {
			marker, length := fenceRun(trimmed)
			switch {
			case fence == 0 && length >= 3 && (marker != '`' || !strings.ContainsRune(trimmed[length:], '`')):
				fence = marker
				fenceLength = length
			case fence != 0 && marker == fence && length >= fenceLength && strings.TrimSpace(trimmed[length:]) == "":
				fence = 0
				fenceLength = 0
			case fence == 0 && sectionHeading.MatchString(line):
				return offset
			}
		}
		offset = nextOffset
	}
	return -1
}

func fenceRun(line string) (byte, int) {
	if line == "" || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	marker := line[0]
	length := 1
	for length < len(line) && line[length] == marker {
		length++
	}
	return marker, length
}
