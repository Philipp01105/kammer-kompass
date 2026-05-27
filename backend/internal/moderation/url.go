package moderation

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrBlockedURLScheme = errors.New("blocked url scheme")
	ErrTooManyLinks     = errors.New("too many links")
)

var httpLinkRegex = regexp.MustCompile(`https?://[^\s]+`)

// CheckURLSafety checks if the text contains blocked URL schemes or too many links
// TODO: extend list of blocked schemes
func CheckURLSafety(text string, maxLinks int) error {
	t := strings.ToLower(text)
	if strings.Contains(t, "javascript:") ||
		strings.Contains(t, "data:") ||
		strings.Contains(t, "file:") ||
		strings.Contains(t, "vbscript:") {
		return ErrBlockedURLScheme
	}

	if maxLinks > 0 && len(httpLinkRegex.FindAllString(text, -1)) > maxLinks {
		return ErrTooManyLinks
	}

	return nil
}
