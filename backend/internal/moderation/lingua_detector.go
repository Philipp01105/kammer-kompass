package moderation

import (
	"strings"

	"github.com/pemistahl/lingua-go"
)

type LinguaDetector struct {
	detector lingua.LanguageDetector
}

func NewLinguaDetector() LinguaDetector {
	d := lingua.NewLanguageDetectorBuilder().
		FromAllLanguages().
		WithMinimumRelativeDistance(0.0).
		Build()
	return LinguaDetector{detector: d}
}

func (l LinguaDetector) DetectGermanConfidence(text string) float64 {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	confidences := l.detector.ComputeLanguageConfidenceValues(text)
	for _, c := range confidences {
		if c.Language() == lingua.German {
			return c.Value()
		}
	}
	return 0
}
