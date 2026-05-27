package moderation

func CheckWordFilter(text string, normalizedTerms []string) (blocked bool, matched string) {
	normalized := Normalize(text)
	return ContainsBlockedTerm(normalized, normalizedTerms)
}
