// Package align matches two word sequences, such as the words of an EPUB and
// the words of its audiobook transcript. It finds reliable exact and fuzzy
// anchors first, then aligns the short gaps between them with dynamic
// programming, so that a whole book can be matched without a quadratic table
// over every word.
package align

import "context"

// Result is the compact, backwards-compatible result of Sequences. Match[i]
// is the target index of source word i, or -1 when the word could not be
// matched. Confidence is ebook coverage; use AlignmentResult for the full
// weighted confidence score.
type Result struct {
	Match      []int
	Matched    int
	Total      int
	Confidence float64
}

const (
	scoreMatch    = 3
	scoreMismatch = -3
	scoreGap      = -1
)

const (
	directionDiagonal byte = iota + 1
	directionUp
	directionLeft
)

// Sequences aligns source to target with the default configuration. It is the
// compact API kept for existing callers; new code should use AlignSequences or
// AlignChapters to get coverage, quality and diagnostics.
func Sequences(source, target []string) Result {
	result, _ := AlignSequencesContext(context.Background(), source, target, DefaultConfig(), nil)

	compact := Result{
		Match:      make([]int, len(source)),
		Total:      len(source),
		Confidence: result.EbookCoverage,
	}
	for index := range compact.Match {
		compact.Match[index] = -1
	}
	for _, match := range result.Matches {
		if match.EbookWordIndex < 0 || match.EbookWordIndex >= len(compact.Match) {
			continue
		}
		compact.Match[match.EbookWordIndex] = match.AudioWordIndex
		compact.Matched++
	}
	return compact
}
