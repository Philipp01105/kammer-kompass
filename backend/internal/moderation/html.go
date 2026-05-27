package moderation

import "strings"

// TODO: check if this is enough
var blockedSubstrings = []string{
	"<script",
	"</script",
	"<iframe",
	"javascript:",
	"onerror=",
	"onclick=",
	"style=",
}

func ContainsBlockedHTML(text string) bool {
	t := strings.ToLower(text)
	for _, s := range blockedSubstrings {
		if strings.Contains(t, s) {
			return true
		}
	}
	return false
}
