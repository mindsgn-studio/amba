package align

import (
	"github.com/mindsgn-studio/amba/internal/textutil"
)

// WordSimilarity compares two words on a scale from 0 to 1. Identical words
// score 1, a small spelling difference scores highly, and unrelated words
// score near 0. Empty words never match.
//
// The function normalises its inputs, so "The" and "the." compare equal. It is
// deliberately lexical: it does not know that two different words can mean the
// same thing.
func WordSimilarity(a, b string) float64 {
	a = textutil.Normalize(a)
	b = textutil.Normalize(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}

	left := []rune(a)
	right := []rune(b)
	longest := len(left)
	if len(right) > longest {
		longest = len(right)
	}
	if longest == 0 {
		return 0
	}

	// Very short words carry too little signal for a fuzzy match. "a" and
	// "i", for example, are one edit apart but are never the same word.
	if longest <= 2 {
		return 0
	}

	distance := textutil.Levenshtein(a, b)
	score := 1 - float64(distance)/float64(longest)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// similar reports whether two words are close enough to be considered the
// same word by the alignment.
func similar(a, b string, minimum float64) (float64, bool) {
	score := WordSimilarity(a, b)
	if score < minimum || score <= 0 {
		return score, false
	}
	return score, true
}
