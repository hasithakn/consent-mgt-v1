package utils

import (
	"html"
	"net/url"
	"strings"
	"unicode"
)

func SanitizeString(input string) string {
	trimmed := strings.TrimSpace(input)
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, trimmed)
	return html.EscapeString(cleaned)
}

func IsValidURI(uri string) bool {
	parsed, err := url.Parse(uri)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}
