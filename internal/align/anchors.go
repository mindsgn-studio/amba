package align

import (
	"context"
	"sort"
	"strings"
)

const (
	maxAnchorFrequency = defaultMaxAnchorFrequency
	maxFuzzyPositions  = defaultMaxFuzzyKeyPositions
)

type anchorCandidate struct {
	source int
	target int
	size   int
	score  float64
}

// findAnchorsContext finds exact and fuzzy n-gram anchors that occur in
// increasing order in both sequences. Larger, more distinctive anchors are
// preferred, and overlapping candidates are discarded.
func findAnchorsContext(ctx context.Context, source, target []string, config Config) ([]anchorCandidate, error) {
	if len(source) < 2 || len(target) < 2 {
		return nil, nil
	}

	seen := make(map[[3]int]anchorCandidate)
	add := func(candidate anchorCandidate) {
		if candidate.source < 0 || candidate.target < 0 {
			return
		}
		key := [3]int{candidate.source, candidate.target, candidate.size}
		if existing, ok := seen[key]; ok && existing.score >= candidate.score {
			return
		}
		seen[key] = candidate
	}

	for _, size := range config.AnchorSizes {
		if size > len(source) || size > len(target) {
			continue
		}
		sourceIndex := uniqueShinglesOfSize(source, size)
		targetIndex := uniqueShinglesOfSize(target, size)
		for key, sourceAt := range sourceIndex {
			targetAt, ok := targetIndex[key]
			if !ok {
				continue
			}
			words := source[sourceAt : sourceAt+size]
			if !anchorAllowed(words, size) {
				continue
			}
			add(anchorCandidate{source: sourceAt, target: targetAt, size: size, score: float64(size)})
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	fuzzy, err := fuzzyAnchors(ctx, source, target, config)
	if err != nil {
		return nil, err
	}
	for _, candidate := range fuzzy {
		add(candidate)
	}

	candidates := make([]anchorCandidate, 0, len(seen))
	for _, candidate := range seen {
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].source != candidates[j].source {
			return candidates[i].source < candidates[j].source
		}
		if candidates[i].target != candidates[j].target {
			return candidates[i].target < candidates[j].target
		}
		return candidates[i].score > candidates[j].score
	})

	ordered := weightedIncreasing(candidates)
	return removeOverlaps(ordered), nil
}

func uniqueShinglesOfSize(words []string, size int) map[string]int {
	index := make(map[string]int, len(words))
	duplicate := make(map[string]struct{})

	for i := 0; i+size <= len(words); i++ {
		key := strings.Join(words[i:i+size], " ")
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

// fuzzyAnchors finds n-grams that differ by a few words. It indexes the
// target by distinctive words, so it never compares every possible pair.
func fuzzyAnchors(ctx context.Context, source, target []string, config Config) ([]anchorCandidate, error) {
	targetIndex := make(map[string][]int)
	for _, size := range config.AnchorSizes {
		if size < 3 || size > len(target) {
			continue
		}
		for i := 0; i+size <= len(target); i++ {
			for _, key := range distinctiveWords(target[i : i+size]) {
				if len(targetIndex[key]) >= maxFuzzyPositions {
					continue
				}
				targetIndex[key] = append(targetIndex[key], i)
			}
		}
	}

	sourceCounts := make(map[string]int)
	for _, size := range config.AnchorSizes {
		if size < 3 || size > len(source) {
			continue
		}
		for i := 0; i+size <= len(source); i++ {
			for _, key := range distinctiveWords(source[i : i+size]) {
				sourceCounts[key]++
			}
		}
	}

	var candidates []anchorCandidate
	for _, size := range config.AnchorSizes {
		if size < 3 || size > len(source) || size > len(target) {
			continue
		}
		for i := 0; i+size <= len(source); i++ {
			if i%64 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			words := source[i : i+size]
			if !anchorAllowed(words, size) {
				continue
			}
			keys := distinctiveWords(words)
			if len(keys) == 0 {
				continue
			}

			candidatePositions := make(map[int]struct{})
			for _, key := range keys {
				if sourceCounts[key] > maxAnchorFrequency {
					continue
				}
				for _, targetAt := range targetIndex[key] {
					candidatePositions[targetAt] = struct{}{}
				}
			}

			for targetAt := range candidatePositions {
				if targetAt+size > len(target) {
					continue
				}
				if exactWords(words, target[targetAt:targetAt+size]) {
					continue
				}
				average, minimum, ok := ngramSimilarity(words, target[targetAt:targetAt+size], config)
				if !ok || average < config.FuzzyAnchorSimilarity || minimum < config.FuzzyAnchorWordMinimum {
					continue
				}
				candidates = append(candidates, anchorCandidate{
					source: i,
					target: targetAt,
					size:   size,
					score:  float64(size) * average * 0.95,
				})
			}
		}
	}
	return candidates, nil
}

func exactWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ngramSimilarity(a, b []string, config Config) (average, minimum float64, ok bool) {
	if len(a) != len(b) || len(a) == 0 {
		return 0, 0, false
	}
	minimum = 1
	for i := range a {
		score := WordSimilarity(a[i], b[i])
		if score < config.MinWordSimilarity {
			return 0, 0, false
		}
		average += score
		if score < minimum {
			minimum = score
		}
	}
	return average / float64(len(a)), minimum, true
}

// distinctiveWords returns up to three content words that can key an n-gram
// index. Generic words such as "the" and "and" are ignored.
func distinctiveWords(words []string) []string {
	type candidate struct {
		word  string
		score int
	}
	var candidates []candidate
	seen := make(map[string]struct{})
	for _, word := range words {
		if len(word) < 3 || genericWords[word] {
			continue
		}
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		candidates = append(candidates, candidate{word: word, score: len(word)})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	limit := len(candidates)
	if limit > 3 {
		limit = 3
	}
	keys := make([]string, limit)
	for i := 0; i < limit; i++ {
		keys[i] = candidates[i].word
	}
	return keys
}

func anchorAllowed(words []string, size int) bool {
	if size <= 1 {
		return false
	}
	key := strings.Join(words, " ")
	if _, generic := genericPhrases[key]; generic {
		return false
	}
	if size <= 3 {
		distinctive := false
		for _, word := range words {
			if !genericWords[word] {
				distinctive = true
				break
			}
		}
		if !distinctive {
			return false
		}
	}
	return true
}

// weightedIncreasing returns the highest-scoring subsequence whose target
// positions increase. The input is sorted by source position.
func weightedIncreasing(candidates []anchorCandidate) []anchorCandidate {
	if len(candidates) == 0 {
		return nil
	}
	maxTarget := 0
	for _, candidate := range candidates {
		if candidate.target > maxTarget {
			maxTarget = candidate.target
		}
	}

	treeScore := make([]float64, maxTarget+2)
	treeIndex := make([]int, maxTarget+2)
	for i := range treeIndex {
		treeIndex[i] = -1
	}

	previous := make([]int, len(candidates))
	score := make([]float64, len(candidates))
	for i := range previous {
		previous[i] = -1
	}

	bestIndex := -1
	bestScore := 0.0
	for i, candidate := range candidates {
		best, from := queryTree(treeScore, treeIndex, candidate.target)
		score[i] = best + candidate.score
		previous[i] = from
		updateTree(treeScore, treeIndex, candidate.target+1, score[i], i)
		if score[i] > bestScore {
			bestScore = score[i]
			bestIndex = i
		}
	}
	if bestIndex < 0 {
		return nil
	}

	var ordered []anchorCandidate
	for at := bestIndex; at >= 0; at = previous[at] {
		ordered = append(ordered, candidates[at])
	}
	for left, right := 0, len(ordered)-1; left < right; left, right = left+1, right-1 {
		ordered[left], ordered[right] = ordered[right], ordered[left]
	}
	return ordered
}

func queryTree(scores []float64, indexes []int, position int) (float64, int) {
	best := 0.0
	index := -1
	for i := position; i > 0; i -= i & -i {
		if scores[i] > best {
			best = scores[i]
			index = indexes[i]
		}
	}
	return best, index
}

func updateTree(scores []float64, indexes []int, position int, score float64, index int) {
	for i := position; i < len(scores); i += i & -i {
		if score > scores[i] {
			scores[i] = score
			indexes[i] = index
		}
	}
}

func removeOverlaps(candidates []anchorCandidate) []anchorCandidate {
	if len(candidates) == 0 {
		return nil
	}
	kept := make([]anchorCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		skip := false
		for len(kept) > 0 {
			last := kept[len(kept)-1]
			if candidate.source >= last.source+last.size && candidate.target >= last.target+last.size {
				break
			}
			if candidate.score > last.score {
				kept = kept[:len(kept)-1]
				continue
			}
			skip = true
			break
		}
		if !skip {
			kept = append(kept, candidate)
		}
	}
	return kept
}

var genericWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
	"if": true, "then": true, "of": true, "to": true, "in": true, "on": true,
	"at": true, "for": true, "with": true, "from": true, "by": true, "as": true,
	"is": true, "was": true, "were": true, "be": true, "been": true, "being": true,
	"am": true, "are": true, "it": true, "its": true, "he": true, "she": true,
	"they": true, "them": true, "his": true, "her": true, "their": true, "we": true,
	"you": true, "i": true, "me": true, "my": true, "your": true, "our": true,
	"this": true, "that": true, "these": true, "those": true, "there": true,
	"here": true, "not": true, "no": true, "yes": true, "so": true, "do": true,
	"does": true, "did": true, "has": true, "have": true, "had": true, "will": true,
	"would": true, "can": true, "could": true, "should": true, "may": true,
	"might": true, "must": true, "said": true, "say": true, "says": true,
}

var genericPhrases = map[string]bool{
	"and then": true, "in the end": true, "he said": true, "she said": true,
	"they said": true, "of the": true, "in the": true, "on the": true,
	"to the": true, "at the": true, "it was": true, "there was": true,
	"and the": true, "that the": true, "with the": true, "for the": true,
}
