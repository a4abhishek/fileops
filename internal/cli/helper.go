package cli

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// capitalizeFirst properly capitalizes the first letter of a string using Unicode-aware title casing
// This replaces the deprecated strings.Title with proper locale-aware functionality
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	// Use English language tag for title casing
	caser := cases.Title(language.English)
	return caser.String(s)
}
