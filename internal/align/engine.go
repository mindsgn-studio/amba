package align

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// AlgorithmVersion identifies the alignment algorithm and the meaning of its
// cached results. Change it whenever the algorithm changes.
const AlgorithmVersion = 2

const (
	defaultMinConfidence          = 0.80
	defaultMinEbookCoverage       = 0.80
	defaultMinQuality             = 0.80
	defaultMinAudioCoverage       = 0.0
	defaultMinSequentialEvidence  = 3
	defaultMinWordSimilarity      = 0.74
	defaultFuzzyAnchorSimilarity  = 0.84
	defaultFuzzyAnchorWordMinimum = 0.68
	defaultMaxDPCells             = 1_500_000
	defaultGreedyWindow           = 128
	defaultMaxAnchorFrequency     = 6
	defaultMaxFuzzyKeyPositions   = 12
)

// Config controls how strict alignment and acceptance are. The zero value is
// not useful; use DefaultConfig and change the fields you need.
type Config struct {
	// MinConfidence, MinEbookCoverage and MinQuality are the acceptance
	// thresholds. They are alignment-quality scores, not calibrated
	// probabilities.
	MinConfidence    float64
	MinEbookCoverage float64
	MinQuality       float64
	// MinAudioCoverage is optional. The default is zero because an audiobook
	// may contain introductions or acknowledgements that the ebook does not.
	MinAudioCoverage float64
	// MinSequentialEvidence is the smallest number of matched words needed
	// to trust a chapter or book. Short chapters use their own length.
	MinSequentialEvidence int
	// MinWordSimilarity is the smallest word score that counts as a match.
	MinWordSimilarity float64
	// FuzzyAnchorSimilarity is the average score needed for a fuzzy anchor.
	FuzzyAnchorSimilarity float64
	// FuzzyAnchorWordMinimum is the smallest per-word score allowed inside a
	// fuzzy anchor.
	FuzzyAnchorWordMinimum float64
	// MaxDPCells bounds the dynamic-programming table. Larger gaps use a
	// bounded greedy scan instead.
	MaxDPCells int
	// GreedyWindow is how far the greedy fallback looks ahead.
	GreedyWindow int
	// AnchorSizes are tried from largest to smallest.
	AnchorSizes []int
}

// DefaultConfig is the production alignment configuration.
func DefaultConfig() Config {
	return Config{
		MinConfidence:          defaultMinConfidence,
		MinEbookCoverage:       defaultMinEbookCoverage,
		MinQuality:             defaultMinQuality,
		MinAudioCoverage:       defaultMinAudioCoverage,
		MinSequentialEvidence:  defaultMinSequentialEvidence,
		MinWordSimilarity:      defaultMinWordSimilarity,
		FuzzyAnchorSimilarity:  defaultFuzzyAnchorSimilarity,
		FuzzyAnchorWordMinimum: defaultFuzzyAnchorWordMinimum,
		MaxDPCells:             defaultMaxDPCells,
		GreedyWindow:           defaultGreedyWindow,
		AnchorSizes:            []int{5, 4, 3, 2},
	}
}

// Canonical returns the configuration with defaults applied. It is useful
// for cache keys and diagnostics.
func (c Config) Canonical() Config { return c.normalized() }

func (c Config) normalized() Config {
	defaults := DefaultConfig()
	if c.MinConfidence <= 0 {
		c.MinConfidence = defaults.MinConfidence
	}
	if c.MinEbookCoverage <= 0 {
		c.MinEbookCoverage = defaults.MinEbookCoverage
	}
	if c.MinQuality <= 0 {
		c.MinQuality = defaults.MinQuality
	}
	if c.MinAudioCoverage < 0 {
		c.MinAudioCoverage = defaults.MinAudioCoverage
	}
	if c.MinSequentialEvidence <= 0 {
		c.MinSequentialEvidence = defaults.MinSequentialEvidence
	}
	if c.MinWordSimilarity <= 0 {
		c.MinWordSimilarity = defaults.MinWordSimilarity
	}
	if c.FuzzyAnchorSimilarity <= 0 {
		c.FuzzyAnchorSimilarity = defaults.FuzzyAnchorSimilarity
	}
	if c.FuzzyAnchorWordMinimum <= 0 {
		c.FuzzyAnchorWordMinimum = defaults.FuzzyAnchorWordMinimum
	}
	if c.MaxDPCells <= 0 {
		c.MaxDPCells = defaults.MaxDPCells
	}
	if c.GreedyWindow <= 0 {
		c.GreedyWindow = defaults.GreedyWindow
	}
	if len(c.AnchorSizes) == 0 {
		c.AnchorSizes = defaults.AnchorSizes
	}
	sizes := make([]int, 0, len(c.AnchorSizes))
	for _, size := range c.AnchorSizes {
		if size >= 2 {
			sizes = append(sizes, size)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))
	if len(sizes) == 0 {
		sizes = defaults.AnchorSizes
	}
	c.AnchorSizes = sizes
	return c
}

// WordMatch maps one ebook word to one narration word. AudioStart and AudioEnd
// are seconds in the audiobook timeline; they are filled by callers that have
// timestamps.
type WordMatch struct {
	EbookWordIndex int     `json:"ebookWordIndex"`
	AudioWordIndex int     `json:"audioWordIndex"`
	AudioStart     float64 `json:"audioStart"`
	AudioEnd       float64 `json:"audioEnd"`
	Similarity     float64 `json:"similarity"`
}

// AlignmentResult is the outcome of matching one sequence (a book or a
// chapter) to the narration.
type AlignmentResult struct {
	Accepted      bool        `json:"accepted"`
	Confidence    float64     `json:"confidence"`
	EbookCoverage float64     `json:"ebookCoverage"`
	AudioCoverage float64     `json:"audioCoverage"`
	Quality       float64     `json:"quality"`
	Sequential    int         `json:"sequential"`
	Matches       []WordMatch `json:"matches,omitempty"`
	Diagnostics   []string    `json:"diagnostics,omitempty"`
}

// Range is a half-open span of ebook word indices.
type Range struct {
	Start int
	End   int
}

// ChapterResult is the alignment of one ebook chapter. AudioStart and
// AudioEnd are target word indices, or -1 when the chapter has no match.
type ChapterResult struct {
	Index      int   `json:"index"`
	Range      Range `json:"range"`
	AudioStart int   `json:"audioStart"`
	AudioEnd   int   `json:"audioEnd"`
	AlignmentResult
}

// BookAlignment is the whole-book result and its per-chapter breakdown.
type BookAlignment struct {
	AlignmentResult
	Chapters []ChapterResult `json:"chapters,omitempty"`
}

// AlignSequences aligns source words to target words using the default
// configuration.
func AlignSequences(source, target []string) AlignmentResult {
	result, _ := AlignSequencesContext(context.Background(), source, target, DefaultConfig(), nil)
	return result
}

// AlignTimed aligns source words to target words and copies the target
// timestamps into each match.
func AlignTimed(source, target []string, starts, ends []float64, config Config) AlignmentResult {
	result, _ := AlignTimedContext(context.Background(), source, target, starts, ends, config, nil)
	return result
}

// AlignSequencesContext aligns two sequences and can be cancelled. progress,
// when non-nil, receives a fraction between 0 and 1.
func AlignSequencesContext(
	ctx context.Context,
	source, target []string,
	config Config,
	progress func(float64),
) (AlignmentResult, error) {
	return AlignTimedContext(ctx, source, target, nil, nil, config, progress)
}

// AlignTimedContext is AlignSequencesContext with target timestamps.
func AlignTimedContext(
	ctx context.Context,
	source, target []string,
	starts, ends []float64,
	config Config,
	progress func(float64),
) (AlignmentResult, error) {
	config = config.normalized()
	matches, err := alignWords(ctx, source, target, config, progress)
	if err != nil {
		return AlignmentResult{}, err
	}
	fillTimes(matches, starts, ends)
	return evaluate(matches, len(source), len(target), config, true), nil
}

// AlignChapters aligns a book and evaluates each chapter independently. ranges
// are the half-open ebook word ranges of the chapters.
func AlignChapters(
	source, target []string,
	ranges []Range,
	starts, ends []float64,
	config Config,
) BookAlignment {
	result, _ := AlignChaptersContext(context.Background(), source, target, ranges, starts, ends, config, nil)
	return result
}

// AlignChaptersContext is AlignChapters with cancellation and progress.
func AlignChaptersContext(
	ctx context.Context,
	source, target []string,
	ranges []Range,
	starts, ends []float64,
	config Config,
	progress func(float64),
) (BookAlignment, error) {
	config = config.normalized()
	matches, err := alignWords(ctx, source, target, config, progress)
	if err != nil {
		return BookAlignment{}, err
	}
	fillTimes(matches, starts, ends)

	book := BookAlignment{
		AlignmentResult: evaluate(matches, len(source), len(target), config, true),
	}
	if len(ranges) == 0 && len(source) > 0 {
		ranges = []Range{{Start: 0, End: len(source)}}
	}

	byChapter := make([][]WordMatch, len(ranges))
	for _, match := range matches {
		index := chapterOf(ranges, match.EbookWordIndex)
		if index >= 0 {
			byChapter[index] = append(byChapter[index], match)
		}
	}

	for index, span := range ranges {
		chapterMatches := byChapter[index]
		chapter := ChapterResult{
			Index:      index,
			Range:      span,
			AudioStart: -1,
			AudioEnd:   -1,
		}
		if len(chapterMatches) > 0 {
			chapter.AudioStart = chapterMatches[0].AudioWordIndex
			chapter.AudioEnd = chapterMatches[len(chapterMatches)-1].AudioWordIndex + 1
			for _, match := range chapterMatches {
				if match.AudioWordIndex < chapter.AudioStart {
					chapter.AudioStart = match.AudioWordIndex
				}
				if match.AudioWordIndex+1 > chapter.AudioEnd {
					chapter.AudioEnd = match.AudioWordIndex + 1
				}
			}
		}
		chapter.AlignmentResult = evaluateChapter(chapterMatches, span.End-span.Start, chapter.AudioStart, chapter.AudioEnd, config)
		book.Chapters = append(book.Chapters, chapter)
	}
	return book, nil
}

// alignWords produces the monotonic match list without timestamps.
func alignWords(
	ctx context.Context,
	source, target []string,
	config Config,
	progress func(float64),
) ([]WordMatch, error) {
	if len(source) == 0 || len(target) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	reportProgress(progress, 0.05)
	anchors, err := findAnchorsContext(ctx, source, target, config)
	if err != nil {
		return nil, err
	}
	reportProgress(progress, 0.20)

	matchTarget := make([]int, len(source))
	matchScore := make([]float64, len(source))
	for i := range matchTarget {
		matchTarget[i] = -1
	}

	sourceStart, targetStart := 0, 0
	for anchorIndex, anchor := range anchors {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := alignGap(ctx, matchTarget, matchScore, source, target, sourceStart, anchor.source, targetStart, anchor.target, config); err != nil {
			return nil, err
		}
		for k := 0; k < anchor.size; k++ {
			si, ti := anchor.source+k, anchor.target+k
			if si >= len(source) || ti >= len(target) {
				continue
			}
			if score, ok := similar(source[si], target[ti], config.MinWordSimilarity); ok {
				matchTarget[si] = ti
				matchScore[si] = score
			}
		}
		sourceStart = anchor.source + anchor.size
		targetStart = anchor.target + anchor.size
		if len(anchors) > 0 {
			reportProgress(progress, 0.20+0.75*float64(anchorIndex+1)/float64(len(anchors)))
		}
	}
	if err := alignGap(ctx, matchTarget, matchScore, source, target, sourceStart, len(source), targetStart, len(target), config); err != nil {
		return nil, err
	}

	matches := make([]WordMatch, 0, len(source))
	for sourceIndex, targetIndex := range matchTarget {
		if targetIndex < 0 {
			continue
		}
		matches = append(matches, WordMatch{
			EbookWordIndex: sourceIndex,
			AudioWordIndex: targetIndex,
			Similarity:     matchScore[sourceIndex],
		})
	}
	reportProgress(progress, 1)
	return matches, nil
}

func alignGap(
	ctx context.Context,
	matchTarget []int,
	matchScore []float64,
	source, target []string,
	sourceStart, sourceEnd, targetStart, targetEnd int,
	config Config,
) error {
	if sourceStart >= sourceEnd || targetStart >= targetEnd {
		return nil
	}
	matches, scores, err := alignSegmentFuzzy(ctx, source[sourceStart:sourceEnd], target[targetStart:targetEnd], config)
	if err != nil {
		return err
	}
	for index, targetIndex := range matches {
		if targetIndex < 0 {
			continue
		}
		matchTarget[sourceStart+index] = targetStart + targetIndex
		matchScore[sourceStart+index] = scores[index]
	}
	return nil
}

func fillTimes(matches []WordMatch, starts, ends []float64) {
	if len(starts) == 0 {
		return
	}
	for index := range matches {
		target := matches[index].AudioWordIndex
		if target >= 0 && target < len(starts) {
			matches[index].AudioStart = starts[target]
		}
		if target >= 0 && target < len(ends) {
			matches[index].AudioEnd = ends[target]
		}
	}
}

func chapterOf(ranges []Range, sourceIndex int) int {
	for index, span := range ranges {
		if sourceIndex >= span.Start && sourceIndex < span.End {
			return index
		}
	}
	return -1
}

func evaluate(matches []WordMatch, sourceLen, targetLen int, config Config, book bool) AlignmentResult {
	result := AlignmentResult{Matches: matches}
	if sourceLen <= 0 {
		result.Diagnostics = append(result.Diagnostics, "the ebook has no words to match")
		return result
	}

	uniqueTargets := make(map[int]struct{}, len(matches))
	similarity := 0.0
	for _, match := range matches {
		if match.AudioWordIndex >= 0 && match.AudioWordIndex < targetLen {
			uniqueTargets[match.AudioWordIndex] = struct{}{}
		}
		similarity += match.Similarity
	}

	result.Sequential = len(matches)
	result.EbookCoverage = clamp(float64(len(matches)) / float64(sourceLen))
	if targetLen > 0 {
		result.AudioCoverage = clamp(float64(len(uniqueTargets)) / float64(targetLen))
	}
	if len(matches) > 0 {
		average := similarity / float64(len(matches))
		expected := sourceLen
		if targetLen < expected {
			expected = targetLen
		}
		if expected > 0 {
			result.Quality = clamp(average * math.Min(1, float64(len(matches))/float64(expected)))
		}
	}
	result.Confidence = clamp(0.40*result.EbookCoverage + 0.30*result.AudioCoverage + 0.30*result.Quality)
	result.Accepted = accept(&result, sourceLen, config)
	if book {
		result.Diagnostics = append(result.Diagnostics, describe(result, "book", config)...)
	}
	return result
}

func evaluateChapter(matches []WordMatch, sourceLen, audioStart, audioEnd int, config Config) AlignmentResult {
	result := AlignmentResult{Matches: matches}
	if sourceLen <= 0 {
		result.Diagnostics = append(result.Diagnostics, "the chapter has no words to match")
		return result
	}
	if len(matches) == 0 {
		result.Diagnostics = append(result.Diagnostics, "no words in this chapter matched the narration")
		return result
	}

	uniqueTargets := make(map[int]struct{}, len(matches))
	similarity := 0.0
	span := audioEnd - audioStart
	if span <= 0 {
		span = len(matches)
	}
	for _, match := range matches {
		uniqueTargets[match.AudioWordIndex] = struct{}{}
		similarity += match.Similarity
	}

	result.Sequential = len(matches)
	result.EbookCoverage = clamp(float64(len(matches)) / float64(sourceLen))
	result.AudioCoverage = clamp(float64(len(uniqueTargets)) / float64(span))
	average := similarity / float64(len(matches))
	expected := sourceLen
	if span < expected {
		expected = span
	}
	if expected > 0 {
		result.Quality = clamp(average * math.Min(1, float64(len(matches))/float64(expected)))
	}
	result.Confidence = clamp(0.40*result.EbookCoverage + 0.30*result.AudioCoverage + 0.30*result.Quality)
	result.Accepted = accept(&result, sourceLen, config)
	result.Diagnostics = append(result.Diagnostics, describe(result, "chapter", config)...)
	return result
}

func accept(result *AlignmentResult, sourceLen int, config Config) bool {
	requiredEvidence := config.MinSequentialEvidence
	if sourceLen < requiredEvidence {
		requiredEvidence = sourceLen
	}
	ok := true
	if result.Confidence < config.MinConfidence {
		ok = false
	}
	if result.EbookCoverage < config.MinEbookCoverage {
		ok = false
	}
	if result.Quality < config.MinQuality {
		ok = false
	}
	if result.AudioCoverage < config.MinAudioCoverage {
		ok = false
	}
	if result.Sequential < requiredEvidence {
		ok = false
	}
	return ok
}

func describe(result AlignmentResult, subject string, config Config) []string {
	var diagnostics []string
	if result.Confidence < config.MinConfidence {
		diagnostics = append(diagnostics, fmt.Sprintf("%s confidence %.0f%% is below the %.0f%% acceptance threshold", subject, result.Confidence*100, config.MinConfidence*100))
	}
	if result.EbookCoverage < config.MinEbookCoverage {
		diagnostics = append(diagnostics, fmt.Sprintf("%s ebook coverage %.0f%% is below the %.0f%% threshold", subject, result.EbookCoverage*100, config.MinEbookCoverage*100))
	}
	if result.Quality < config.MinQuality {
		diagnostics = append(diagnostics, fmt.Sprintf("%s alignment quality %.0f%% is below the %.0f%% threshold", subject, result.Quality*100, config.MinQuality*100))
	}
	if result.AudioCoverage < config.MinAudioCoverage {
		diagnostics = append(diagnostics, fmt.Sprintf("%s audio coverage %.0f%% is below the %.0f%% threshold", subject, result.AudioCoverage*100, config.MinAudioCoverage*100))
	}
	if result.Sequential == 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("%s has no reliable word matches", subject))
	}
	return diagnostics
}

func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func reportProgress(progress func(float64), fraction float64) {
	if progress != nil {
		progress(clamp(fraction))
	}
}
