package syncer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func createStorageClassName(origName string) (string, error) {
	// Remove accents / diacritics
	nonSpacingMarks := runes.In(unicode.Mn)
	t := transform.Chain(norm.NFD, runes.Remove(nonSpacingMarks), norm.NFC)
	name, _, err := transform.String(t, origName)
	if err != nil {
		return "", err
	}

	// Replace non alphanumeric characters (except .) by a space
	nonAlpha := regexp.MustCompile("[^a-zA-Z0-9.]+")
	name = nonAlpha.ReplaceAllString(name, " ")

	// Use lowercase
	name = strings.ToLower(name)

	// Trim whitespaces
	name = strings.TrimSpace(name)

	// Replace whitespaces by a single dash
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, "-")

	// Truncate
	if len(name) > 253 {
		name = name[:253]
	}

	// Remove trailing and leading "." and "-"
	name = strings.TrimFunc(name, func(r rune) bool { return r == '.' || r == '-' })

	// Return an error if the resulting name is empty
	if len(name) == 0 {
		return "", fmt.Errorf("%s transformed to an empty name", origName)
	}

	return name, nil
}
