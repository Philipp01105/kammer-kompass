package moderation

import "strings"

// TODO: extend
var germanStopwords = []string{
	"der", "die", "das", "und", "oder", "ist", "bei", "für", "mit", "nicht",
	"eine", "einer", "einem", "den", "dem", "im", "am", "zur", "zum",
}

func DetectGermanConfidence(text string, detector LanguageDetector) float64 {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	conf := detector.DetectGermanConfidence(text)
	if conf >= 0.70 {
		return conf
	}

	if len([]rune(text)) < 40 && containsAnyStopword(text) {
		if conf < 0.70 {
			return 0.70
		}
	}
	return conf
}

func containsAnyStopword(text string) bool {
	t := " " + strings.ToLower(text) + " "
	for _, w := range germanStopwords {
		if strings.Contains(t, " "+w+" ") {
			return true
		}
	}
	return false
}
