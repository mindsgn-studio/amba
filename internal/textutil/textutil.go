// Package textutil normalises words so that a printed book and a speech
// transcript can be compared.
package textutil

import (
	"strings"
	"unicode"
)

// Words splits text into whitespace-separated words.
func Words(text string) []string {
	return strings.Fields(text)
}

// Normalize reduces a word to the form used for alignment: lower case, with
// punctuation removed, while apostrophes are kept so that contractions stay
// one word. An ampersand becomes "and", which is how narrators usually read it.
func Normalize(word string) string {
	var b strings.Builder
	b.Grow(len(word))

	for _, r := range strings.ToLower(word) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '\'' || r == '\u2019' || r == '\u2018' || r == '`':
			b.WriteByte('\'')
		case r == '&':
			b.WriteString("and")
		}
	}

	return b.String()
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
