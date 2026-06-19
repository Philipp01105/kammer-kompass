package moderation

import "strings"

// blockedSubstrings is a blocklist of dangerous HTML patterns.
// Covers common XSS vectors but is not a substitute for a proper HTML parser/sanitizer.
// All checks are applied after lowercasing the input.
var blockedSubstrings = []string{
	// Script elements
	"<script", "</script",

	// Frame/embed elements
	"<iframe", "<frame", "<embed", "<object", "<applet",

	// URL schemes
	"javascript:", "vbscript:", "data:",

	// Event handlers — on* attributes
	"onerror=", "onclick=", "onload=", "onmouseover=", "onfocus=",
	"onblur=", "onchange=", "onsubmit=", "onkeydown=", "onkeyup=",
	"onkeypress=", "onmousedown=", "onmouseup=", "onmousemove=",
	"ondblclick=", "oncontextmenu=", "ondrag=", "ondrop=",
	"oninput=", "onpaste=", "oncut=", "oncopy=",
	"onanimationstart=", "onanimationend=", "ontransitionend=",
	"onpointerdown=", "onpointerup=", "onpointermove=",

	// Dangerous attributes
	"style=", "formaction=", "srcdoc=", "xlink:href=",

	// SVG-specific vectors
	"<svg", "<math",
}

// ContainsBlockedHTML returns true if text contains any known XSS vector.
// Note: this is a heuristic blocklist — use a proper HTML allowlist sanitizer
// for full protection in contexts that render user content as HTML.
func ContainsBlockedHTML(text string) bool {
	t := strings.ToLower(text)
	for _, s := range blockedSubstrings {
		if strings.Contains(t, s) {
			return true
		}
	}
	return false
}
