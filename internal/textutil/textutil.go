// Package textutil normalises words so that a printed book and a speech
// transcript can be compared.
package textutil

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Words splits text into whitespace-separated words.
func Words(text string) []string {
	return strings.Fields(text)
}

// Normalize reduces a word to the form used for matching: lower case, with
// punctuation removed, while apostrophes are kept so that contractions stay
// one word. Typographic apostrophes and quotation marks are folded to their
// ASCII forms, Unicode is normalised to NFC, and an ampersand becomes "and",
// which is how narrators usually read it.
//
// The result is for comparison only. Callers must keep the original text for
// display.
func Normalize(word string) string {
	word = norm.NFC.String(word)

	var b strings.Builder
	b.Grow(len(word))

	for _, r := range strings.ToLower(word) {
		switch {
		case r == '\'' || r == '\u2019' || r == '\u2018' || r == '`' || r == '\u02bc':
			b.WriteByte('\'')
		case r == '&':
			b.WriteString("and")
		case r == '\u00a0' || r == '\u2007' || r == '\u202f':
			// Non-breaking spaces separate words in some EPUBs.
			b.WriteByte(' ')
		case r == '\u2010' || r == '\u2011' || r == '\u2012' || r == '\u2013' || r == '\u2014':
			// Dashes and hyphens are not part of a spoken word. Keeping them
			// out makes "well-known" and "wellknown" comparable.
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}

	return strings.TrimSpace(b.String())
}

// IsWord reports whether a token contains at least one letter or digit.
func IsWord(token string) bool {
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// Levenshtein returns the edit distance between two strings, measured in
// runes. It uses one row of memory.
func Levenshtein(a, b string) int {
	left := []rune(a)
	right := []rune(b)
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}
	if len(left) > len(right) {
		left, right = right, left
	}

	previous := make([]int, len(left)+1)
	current := make([]int, len(left)+1)
	for i := range previous {
		previous[i] = i
	}

	for j := 1; j <= len(right); j++ {
		current[0] = j
		for i := 1; i <= len(left); i++ {
			cost := 1
			if left[i-1] == right[j-1] {
				cost = 0
			}
			deletion := previous[i] + 1
			insertion := current[i-1] + 1
			substitution := previous[i-1] + cost
			best := deletion
			if insertion < best {
				best = insertion
			}
			if substitution < best {
				best = substitution
			}
			current[i] = best
		}
		previous, current = current, previous
	}
	return previous[len(left)]
}
