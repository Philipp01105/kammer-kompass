package moderation

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func Normalize(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	text = norm.NFC.String(text)
	text = strings.ToLower(text)

	var b strings.Builder
	b.Grow(len(text))

	wasSpace := false
	for _, r := range text {
		switch r {
		case '1':
			r = 'i'
		case '3':
			r = 'e'
		case '4':
			r = 'a'
		case '0':
			r = 'o'
		case '@':
			r = 'a'
		}

		if unicode.IsSpace(r) {
			if wasSpace {
				continue
			}
			wasSpace = true
			b.WriteByte(' ')
			continue
		}

		wasSpace = false
		b.WriteRune(r)
	}

	return strings.TrimSpace(b.String())
}
