package exponential

import (
	"strings"
	"unicode"
)

// Slugify converts a string into a URL/branch-friendly slug.
// Lowercase, non-alphanumeric runs replaced with hyphens, truncated
// to ~50 characters on a word boundary.
func Slugify(s string) string {
	var b strings.Builder
	prev := '-'
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prev = r
		} else if prev != '-' {
			b.WriteByte('-')
			prev = '-'
		}
	}

	slug := strings.Trim(b.String(), "-")

	if len(slug) <= 50 {
		return slug
	}

	// Truncate on a hyphen boundary
	cut := slug[:50]
	if i := strings.LastIndex(cut, "-"); i > 20 {
		cut = cut[:i]
	}
	return cut
}
