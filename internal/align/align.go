// Package align matches two word sequences, such as the words of an EPUB and
// the words of its audiobook transcript. It finds reliable anchors first and
// aligns the short gaps between them, so that a whole book can be matched
// without a quadratic table over every word.
package align

import (
	"sort"
	"strings"
)

// Result maps every word of the source sequence to a word of the target
// sequence. Match[i] is the target index of source word i, or -1 when the word
// could not be matched.
type Result struct {
	Match      []int
	Matched    int
	Total      int
	Confidence float64
}

const (
	shingleSize   = 3
	maxDPCells    = 1_500_000
	greedyWindow  = 96
	scoreMatch    = 3
	scoreMismatch = -3
	scoreGap      = -1
)

type pair struct {
	source int
	target int
}

// Sequences aligns source to target. Both sequences must contain the
// normalised, non-empty words to compare.
func Sequences(source, target []string) Result {
	result := Result{
		Match: make([]int, len(source)),
		Total: len(source),
	}
	for i := range result.Match {
		result.Match[i] = -1
	}

	if len(source) == 0 || len(target) == 0 {
		return result
	}

	anchors := findAnchors(source, target)

	// The words before the first anchor, between anchors, and after the last
	// one are aligned separately. Anchors are exact matches, so they are
	// mapped directly.
	sourceStart, targetStart := 0, 0
	for _, anchor := range anchors {
		alignInto(result.Match, source, target, sourceStart, anchor.source, targetStart, anchor.target)
		for k := 0; k < shingleSize; k++ {
			result.Match[anchor.source+k] = anchor.target + k
		}
		sourceStart = anchor.source + shingleSize
		targetStart = anchor.target + shingleSize
	}
	alignInto(result.Match, source, target, sourceStart, len(source), targetStart, len(target))

	// Match holds exact matches only. Deciding how to show a word that did not
	// match is the reader's job: readalong gives it the place of a neighbour so
	// highlighting keeps moving, without claiming the word was found.
	for _, index := range result.Match {
		if index >= 0 {
			result.Matched++
		}
	}
	if result.Total > 0 {
		result.Confidence = float64(result.Matched) / float64(result.Total)
	}
	return result
}

// alignInto aligns the segments source[sourceStart:sourceEnd] and
// target[targetStart:targetEnd], writing target indices into match.
func alignInto(match []int, source, target []string, sourceStart, sourceEnd, targetStart, targetEnd int) {
	n := sourceEnd - sourceStart
	m := targetEnd - targetStart
	if n <= 0 || m <= 0 {
		return
	}

	segmentMatch := alignSegment(source[sourceStart:sourceEnd], target[targetStart:targetEnd])
	for i, index := range segmentMatch {
		if index >= 0 {
			match[sourceStart+i] = targetStart + index
		}
	}
}

// findAnchors returns non-overlapping shingles that occur exactly once in both
// sequences, in increasing order in both.
func findAnchors(source, target []string) []pair {
	if len(source) < shingleSize || len(target) < shingleSize {
		return nil
	}

	sourceIndex := uniqueShingles(source)
	targetIndex := uniqueShingles(target)

	candidates := make([]pair, 0, len(sourceIndex))
	for key, sourceAt := range sourceIndex {
		if targetAt, ok := targetIndex[key]; ok {
			candidates = append(candidates, pair{source: sourceAt, target: targetAt})
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].source < candidates[j].source
	})

	ordered := longestIncreasing(candidates)
	if len(ordered) == 0 {
		return nil
	}

	// Keep anchors that do not overlap, so that the segments between them are
	// well defined.
	kept := ordered[:0]
	lastSource, lastTarget := -1, -1
	for _, anchor := range ordered {
		if anchor.source < lastSource+shingleSize || anchor.target < lastTarget+shingleSize {
			continue
		}
		kept = append(kept, anchor)
		lastSource = anchor.source
		lastTarget = anchor.target
	}
	return kept
}

// uniqueShingles maps each shingle that occurs exactly once to its position.
func uniqueShingles(words []string) map[string]int {
	index := make(map[string]int, len(words))
	duplicate := make(map[string]struct{})

	for i := 0; i+shingleSize <= len(words); i++ {
		key := strings.Join(words[i:i+shingleSize], " ")
		if _, exists := index[key]; exists {
			duplicate[key] = struct{}{}
			continue
		}
		index[key] = i
	}
	for key := range duplicate {
		delete(index, key)
	}
	return index
}

// longestIncreasing returns the longest subsequence of pairs whose target
// positions increase. The input is sorted by source position.
func longestIncreasing(pairs []pair) []pair {
	if len(pairs) == 0 {
		return nil
	}

	tails := make([]int, 0, len(pairs))
	previous := make([]int, len(pairs))
	for i := range previous {
		previous[i] = -1
	}

	for i, candidate := range pairs {
		position := sort.Search(len(tails), func(k int) bool {
			return pairs[tails[k]].target >= candidate.target
		})
		if position > 0 {
			previous[i] = tails[position-1]
		}
		if position == len(tails) {
			tails = append(tails, i)
		} else {
			tails[position] = i
		}
	}

	ordered := make([]pair, len(tails))
	at := tails[len(tails)-1]
	for i := len(ordered) - 1; i >= 0; i-- {
		ordered[i] = pairs[at]
		at = previous[at]
	}
	return ordered
}

// alignSegment aligns two short segments with dynamic programming, or with a
// greedy scan when the segments are too large to justify a full table.
func alignSegment(source, target []string) []int {
	match := make([]int, len(source))
	for i := range match {
		match[i] = -1
	}
	if len(source) == 0 || len(target) == 0 {
		return match
	}

	if int64(len(source))*int64(len(target)) > maxDPCells {
		return alignGreedy(source, target)
	}
	return alignDP(source, target)
}

const (
	directionDiagonal byte = iota + 1
	directionUp
	directionLeft
)

func alignDP(source, target []string) []int {
	n, m := len(source), len(target)
	width := m + 1
	scores := make([]int32, (n+1)*width)
	directions := make([]byte, (n+1)*width)

	for i := 1; i <= n; i++ {
		scores[i*width] = int32(i) * scoreGap
		directions[i*width] = directionUp
	}
	for j := 1; j <= m; j++ {
		scores[j] = int32(j) * scoreGap
		directions[j] = directionLeft
	}

	for i := 1; i <= n; i++ {
		row := i * width
		previousRow := (i - 1) * width
		for j := 1; j <= m; j++ {
			diagonalScore := scores[previousRow+j-1] + wordScore(source[i-1], target[j-1])
			upScore := scores[previousRow+j] + scoreGap
			leftScore := scores[row+j-1] + scoreGap

			best := diagonalScore
			direction := directionDiagonal
			if upScore > best {
				best = upScore
				direction = directionUp
			}
			if leftScore > best {
				best = leftScore
				direction = directionLeft
			}
			scores[row+j] = best
			directions[row+j] = direction
		}
	}

	match := make([]int, n)
	for i := range match {
		match[i] = -1
	}

	i, j := n, m
	for i > 0 || j > 0 {
		switch directions[i*width+j] {
		case directionDiagonal:
			if source[i-1] == target[j-1] {
				match[i-1] = j - 1
			}
			i--
			j--
		case directionUp:
			i--
		case directionLeft:
			j--
		default:
			// One of the indices is zero; walk the remaining axis.
			if i > 0 {
				i--
			} else {
				j--
			}
		}
	}
	return match
}

func wordScore(source, target string) int32 {
	if source == target {
		return scoreMatch
	}
	return scoreMismatch
}

// alignGreedy matches each source word to the next equal target word within a
// window. It is used only for segments too large for the dynamic table, which
// are usually front matter that the audiobook omits.
func alignGreedy(source, target []string) []int {
	match := make([]int, len(source))
	for i := range match {
		match[i] = -1
	}

	at := 0
	for i := 0; i < len(source) && at < len(target); i++ {
		limit := at + greedyWindow
		if limit > len(target) {
			limit = len(target)
		}
		for j := at; j < limit; j++ {
			if source[i] == target[j] {
				match[i] = j
				at = j + 1
				break
			}
		}
	}
	return match
}
