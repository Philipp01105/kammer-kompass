package moderation

import (
	"strings"
	"unicode"
)

func TokenizeWords(text string) []string {
	var b strings.Builder
	tokens := make([]string, 0, 32)

	flush := func() {
		if b.Len() == 0 {
			return
		}
		tokens = append(tokens, b.String())
		b.Reset()
	}

	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return tokens
}

func ContainsBlockedTerm(text string, normalizedTerms []string) (bool, string) {
	textTokens := TokenizeWords(text)
	if len(textTokens) == 0 || len(normalizedTerms) == 0 {
		return false, ""
	}

	termTokensList := make([][]string, 0, len(normalizedTerms))
	for _, t := range normalizedTerms {
		termTokens := TokenizeWords(t)
		if len(termTokens) == 0 {
			continue
		}
		termTokensList = append(termTokensList, termTokens)
	}

	for _, termTokens := range termTokensList {
		if containsTokenSequence(textTokens, termTokens) {
			return true, strings.Join(termTokens, " ")
		}
	}

	return false, ""
}

func containsTokenSequence(haystack []string, needle []string) bool {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return false
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
