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

var (
	httpLinkRegex = regexp.MustCompile(`https?://[^\s]+`)
	schemeRegex   = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]{1,31})\s*:`)
)

var blockedSchemes = map[string]struct{}{
	"about":      {},
	"blob":       {},
	"chrome":     {},
	"content":    {},
	"data":       {},
	"file":       {},
	"filesystem": {},
	"ftp":        {},
	"gopher":     {},
	"javascript": {},
	"vbscript":   {},
}

// CheckURLSafety blocks executable/local URL schemes and limits the number of
// public HTTP(S) links allowed in user-submitted text.
func CheckURLSafety(text string, maxLinks int) error {
	for _, match := range schemeRegex.FindAllStringSubmatch(strings.ToLower(text), -1) {
		if len(match) < 2 {
			continue
		}
		if _, blocked := blockedSchemes[strings.TrimSpace(match[1])]; blocked {
			return ErrBlockedURLScheme
		}
	}

	if maxLinks > 0 && len(httpLinkRegex.FindAllString(text, -1)) > maxLinks {
		return ErrTooManyLinks
	}

	return nil
}
