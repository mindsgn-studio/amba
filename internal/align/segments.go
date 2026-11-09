package align

import (
	"context"
	"math"
)

// alignSegmentFuzzy aligns two short segments with dynamic programming, or
// with a bounded greedy scan when the table would be too large. It returns a
// target index and similarity for each source word, or -1 when unmatched.
func alignSegmentFuzzy(ctx context.Context, source, target []string, config Config) ([]int, []float64, error) {
	matches := make([]int, len(source))
	scores := make([]float64, len(source))
	for i := range matches {
		matches[i] = -1
	}
	if len(source) == 0 || len(target) == 0 {
		return matches, scores, nil
	}

	if int64(len(source))*int64(len(target)) > int64(config.MaxDPCells) {
		alignGreedyFuzzy(source, target, config, matches, scores)
		return matches, scores, nil
	}
	if err := alignDPFuzzy(ctx, source, target, config, matches, scores); err != nil {
		return nil, nil, err
	}
	return matches, scores, nil
}

func alignDPFuzzy(
	ctx context.Context,
	source, target []string,
	config Config,
	match []int,
	matchScore []float64,
) error {
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
		if i%64 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		row := i * width
		previousRow := (i - 1) * width
		for j := 1; j <= m; j++ {
			similarity := WordSimilarity(source[i-1], target[j-1])
			diagonal := scores[previousRow+j-1] + wordScoreFuzzy(similarity, config)
			up := scores[previousRow+j] + scoreGap
			left := scores[row+j-1] + scoreGap

			best := diagonal
			direction := directionDiagonal
			if up > best {
				best = up
				direction = directionUp
			}
			if left > best {
				best = left
				direction = directionLeft
			}
			scores[row+j] = best
			directions[row+j] = direction
		}
	}

	i, j := n, m
	for i > 0 || j > 0 {
		switch directions[i*width+j] {
		case directionDiagonal:
			similarity := WordSimilarity(source[i-1], target[j-1])
			if similarity >= config.MinWordSimilarity {
				match[i-1] = j - 1
				matchScore[i-1] = similarity
			}
			i--
			j--
		case directionUp:
			i--
		case directionLeft:
			j--
		default:
			if i > 0 {
				i--
			} else {
				j--
			}
		}
	}
	return nil
}

func wordScoreFuzzy(similarity float64, config Config) int32 {
	if similarity < config.MinWordSimilarity {
		return scoreMismatch
	}
	value := int32(math.Round(similarity * float64(scoreMatch)))
	if value < 1 {
		return 1
	}
	return value
}

// alignGreedyFuzzy matches each source word to the best similar target word in
// a bounded forward window. It is used only when a full table would be too
// large, usually for front matter that the narration omits.
func alignGreedyFuzzy(source, target []string, config Config, match []int, matchScore []float64) {
	at := 0
	for i := 0; i < len(source) && at < len(target); i++ {
		limit := at + config.GreedyWindow
		if limit > len(target) {
			limit = len(target)
		}
		bestAt := -1
		bestScore := 0.0
		for j := at; j < limit; j++ {
			score := WordSimilarity(source[i], target[j])
			if score < config.MinWordSimilarity {
				continue
			}
			if score > bestScore {
				bestScore = score
				bestAt = j
				if score == 1 {
					break
				}
			}
		}
		if bestAt >= 0 {
			match[i] = bestAt
			matchScore[i] = bestScore
			at = bestAt + 1
		}
	}
}

// LocalAlign finds the best matching passage between two sequences using
// Smith-Waterman local alignment. It is useful for locating a chapter in an
// audiobook when the editions have different boundaries, and for diagnosing
// partially matching books. The returned matches are indices into source and
// target.
func LocalAlign(source, target []string, config Config) AlignmentResult {
	config = config.normalized()
	if len(source) == 0 || len(target) == 0 {
		return AlignmentResult{}
	}
	n, m := len(source), len(target)
	width := m + 1
	scores := make([]int32, (n+1)*width)
	directions := make([]byte, (n+1)*width)

	bestScore := int32(0)
	bestI, bestJ := 0, 0
	for i := 1; i <= n; i++ {
		row := i * width
		previousRow := (i - 1) * width
		for j := 1; j <= m; j++ {
			similarity := WordSimilarity(source[i-1], target[j-1])
			diagonal := scores[previousRow+j-1] + wordScoreFuzzy(similarity, config)
			if diagonal < 0 {
				diagonal = 0
			}
			up := scores[previousRow+j] + scoreGap
			if up < 0 {
				up = 0
			}
			left := scores[row+j-1] + scoreGap
			if left < 0 {
				left = 0
			}

			best := diagonal
			direction := directionDiagonal
			if up > best {
				best = up
				direction = directionUp
			}
			if left > best {
				best = left
				direction = directionLeft
			}
			if best == 0 {
				direction = 0
			}
			scores[row+j] = best
			directions[row+j] = direction
			if best > bestScore {
				bestScore = best
				bestI, bestJ = i, j
			}
		}
	}
	if bestScore == 0 {
		return AlignmentResult{}
	}

	var matches []WordMatch
	i, j := bestI, bestJ
	for i > 0 && j > 0 {
		switch directions[i*width+j] {
		case directionDiagonal:
			similarity := WordSimilarity(source[i-1], target[j-1])
			if similarity >= config.MinWordSimilarity {
				matches = append(matches, WordMatch{
					EbookWordIndex: i - 1,
					AudioWordIndex: j - 1,
					Similarity:     similarity,
				})
			}
			i--
			j--
		case directionUp:
			i--
		case directionLeft:
			j--
		default:
			i, j = 0, 0
		}
	}
	for left, right := 0, len(matches)-1; left < right; left, right = left+1, right-1 {
		matches[left], matches[right] = matches[right], matches[left]
	}
	return evaluate(matches, len(source), len(target), config, true)
}
