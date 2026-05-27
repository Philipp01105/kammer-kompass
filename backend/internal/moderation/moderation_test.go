package moderation

import (
	"errors"
	"testing"
)

type fixedDetector float64

func (d fixedDetector) DetectGermanConfidence(string) float64 {
	return float64(d)
}

func TestNormalize(t *testing.T) {
	got := Normalize("  D4s   1st @uch  0kay  ")
	want := "das ist auch okay"
	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestBlockedHTML(t *testing.T) {
	blocked := []string{"<script>alert(1)</script>", "<iframe src=x>", "javascript:alert(1)", "onerror=alert(1)", "onclick=x", "style=color:red"}
	for _, input := range blocked {
		if !ContainsBlockedHTML(input) {
			t.Fatalf("ContainsBlockedHTML(%q) = false, want true", input)
		}
	}
}

func TestURLSafety(t *testing.T) {
	if err := CheckURLSafety("https://a.example http://b.example https://c.example http://d.example", 3); !errors.Is(err, ErrTooManyLinks) {
		t.Fatalf("CheckURLSafety too many links error = %v, want %v", err, ErrTooManyLinks)
	}
	if err := CheckURLSafety("javascript:alert(1)", 3); !errors.Is(err, ErrBlockedURLScheme) {
		t.Fatalf("CheckURLSafety blocked scheme error = %v, want %v", err, ErrBlockedURLScheme)
	}
}

func TestDetectGermanConfidenceUsesShortTextStopwordHeuristic(t *testing.T) {
	got := DetectGermanConfidence("ist bei der IHK", fixedDetector(0.2))
	if got != 0.70 {
		t.Fatalf("DetectGermanConfidence() = %v, want 0.70", got)
	}
}

func TestWordFilter(t *testing.T) {
	blocked, matched := CheckWordFilter("Das ist ein sp4mlink", []string{"spamlink"})
	if !blocked || matched != "spamlink" {
		t.Fatalf("CheckWordFilter() = (%v, %q), want (true, spamlink)", blocked, matched)
	}
}
