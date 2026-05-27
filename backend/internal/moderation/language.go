package moderation

type LanguageDetector interface {
	DetectGermanConfidence(text string) float64
}
