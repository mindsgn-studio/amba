package transcribe

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// Span is a stretch of time inside an audio chunk, in chunk-relative time.
type Span struct {
	Start time.Duration
	End   time.Duration
}

const (
	frameDuration   = 20 * time.Millisecond
	shortPause      = 250 * time.Millisecond
	shortSpeech     = 100 * time.Millisecond
	silenceRatio    = 0.15
	minSignalEnergy = 1e-4
)

// speechSpans finds the parts of a chunk that contain speech. Audiobook
// narration is clean, so an energy threshold relative to the loudest frame
// separates the narrator from pauses well enough to place words in time.
func speechSpans(samples []float32, sampleRate int) []Span {
	if sampleRate <= 0 || len(samples) == 0 {
		return nil
	}

	frameSize := int(float64(sampleRate) * frameDuration.Seconds())
	if frameSize < 1 {
		frameSize = 1
	}
	frames := len(samples) / frameSize
	if frames == 0 {
		return nil
	}

	energies := make([]float64, frames)
	peak := 0.0
	for index := range energies {
		var sum float64
		for _, sample := range samples[index*frameSize : (index+1)*frameSize] {
			sum += float64(sample) * float64(sample)
		}
		energies[index] = math.Sqrt(sum / float64(frameSize))
		if energies[index] > peak {
			peak = energies[index]
		}
	}
	if peak < minSignalEnergy {
		return nil
	}

	threshold := peak * silenceRatio
	var runs []Span
	for index, energy := range energies {
		if energy < threshold {
			continue
		}
		start := time.Duration(float64(index*frameSize) / float64(sampleRate) * float64(time.Second))
		end := time.Duration(float64((index+1)*frameSize) / float64(sampleRate) * float64(time.Second))
		if len(runs) > 0 && start-runs[len(runs)-1].End <= shortPause {
			runs[len(runs)-1].End = end
			continue
		}
		runs = append(runs, Span{Start: start, End: end})
	}

	// Drop isolated blips, which are usually a breath or a page turn.
	filtered := runs[:0]
	for _, run := range runs {
		if run.End-run.Start >= shortSpeech {
			filtered = append(filtered, run)
		}
	}
	return filtered
}

// estimateWords places recognised words in a chunk when the speech model does
// not provide token timestamps. Words are spread over the chunk's speech in
// proportion to their length, so pauses interrupt the highlighting rather
// than stretching the words around them.
func estimateWords(text string, offset, duration time.Duration, spans []Span) []Word {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return nil
	}
	if duration <= 0 {
		duration = time.Duration(len(fields)) * 300 * time.Millisecond
	}

	spans = clampSpans(spans, duration)
	if len(spans) == 0 {
		spans = []Span{{Start: 0, End: duration}}
	}

	var speech time.Duration
	for _, span := range spans {
		speech += span.End - span.Start
	}

	weights := make([]int, len(fields))
	total := 0
	for index, field := range fields {
		weights[index] = 1 + utf8.RuneCountInString(field)
		total += weights[index]
	}

	words := make([]Word, 0, len(fields))
	var before int
	for index, field := range fields {
		start := offset + atSpeechTime(spans, speech, float64(before)/float64(total))
		before += weights[index]
		end := offset + atSpeechTime(spans, speech, float64(before)/float64(total))
		if end <= start {
			end = start + time.Millisecond
		}
		words = append(words, Word{Text: field, Start: start, End: end})
	}
	return words
}

// atSpeechTime turns a fraction of the speech in a chunk into an absolute
// time, skipping the pauses between speech spans.
func atSpeechTime(spans []Span, speech time.Duration, fraction float64) time.Duration {
	remaining := time.Duration(fraction * float64(speech))
	for _, span := range spans {
		length := span.End - span.Start
		if remaining <= length {
			return span.Start + remaining
		}
		remaining -= length
	}
	return spans[len(spans)-1].End
}

func clampSpans(spans []Span, duration time.Duration) []Span {
	var result []Span
	for _, span := range spans {
		if span.End <= 0 || span.Start >= duration {
			continue
		}
		if span.Start < 0 {
			span.Start = 0
		}
		if span.End > duration {
			span.End = duration
		}
		if span.End > span.Start {
			result = append(result, span)
		}
	}
	return result
}
