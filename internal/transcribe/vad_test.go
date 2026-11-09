package transcribe

import (
	"math"
	"testing"
	"time"
)

func toneSamples(sampleRate, seconds int, loud bool) []float32 {
	samples := make([]float32, sampleRate*seconds)
	if !loud {
		return samples
	}
	for index := range samples {
		samples[index] = 0.5 * float32(math.Sin(2*math.Pi*440*float64(index)/float64(sampleRate)))
	}
	return samples
}

func TestSpeechSpansFindsTheSpokenPart(t *testing.T) {
	const rate = 16000
	samples := append(
		append(toneSamples(rate, 1, false), toneSamples(rate, 1, true)...),
		toneSamples(rate, 1, false)...,
	)

	spans := speechSpans(samples, rate)
	if len(spans) != 1 {
		t.Fatalf("speechSpans() = %+v, want one spoken span", spans)
	}
	if !near(spans[0].Start, time.Second) || !near(spans[0].End, 2*time.Second) {
		t.Errorf("speechSpans() = %+v, want about 1s to 2s", spans[0])
	}
}

func TestSpeechSpansOfSilence(t *testing.T) {
	if got := speechSpans(toneSamples(16000, 2, false), 16000); len(got) != 0 {
		t.Errorf("speechSpans(silence) = %+v, want none", got)
	}
	if got := speechSpans(nil, 16000); len(got) != 0 {
		t.Errorf("speechSpans(nil) = %+v, want none", got)
	}
	if got := speechSpans(toneSamples(16000, 1, true), 0); len(got) != 0 {
		t.Errorf("speechSpans(zero rate) = %+v, want none", got)
	}
}

func TestEstimateWordsSkipsPauses(t *testing.T) {
	spans := []Span{{Start: 0, End: time.Second}, {Start: 2 * time.Second, End: 3 * time.Second}}
	words := estimateWords("one two three four", 5*time.Second, 3*time.Second, spans)

	if len(words) != 4 {
		t.Fatalf("estimateWords() = %+v, want four words", words)
	}
	if !near(words[0].Start, 5*time.Second) {
		t.Errorf("first word starts at %v, want 5s", words[0].Start)
	}
	if !near(words[3].End, 8*time.Second) {
		t.Errorf("last word ends at %v, want 8s", words[3].End)
	}
	for index, word := range words {
		if word.End <= word.Start {
			t.Errorf("word %d has no duration: %+v", index, word)
		}
		if index > 0 && word.Start < words[index-1].Start {
			t.Errorf("word %d starts before the previous word: %+v", index, word)
		}
		// The pause is 6s–7s in book time. A word may be heard across it, but
		// no word should begin in the middle of it.
		if word.Start > 6*time.Second && word.Start < 7*time.Second {
			t.Errorf("word %d starts during the pause: %+v", index, word)
		}
	}
}

func TestEstimateWordsWithoutSpans(t *testing.T) {
	words := estimateWords("one two", time.Second, 4*time.Second, nil)
	if len(words) != 2 {
		t.Fatalf("estimateWords() = %+v, want two words", words)
	}
	if !near(words[0].Start, time.Second) || !near(words[1].End, 5*time.Second) {
		t.Errorf("estimateWords() = %+v, want the whole chunk covered", words)
	}
}

func TestEstimateWordsEmpty(t *testing.T) {
	if got := estimateWords("   ", 0, time.Second, nil); len(got) != 0 {
		t.Errorf("estimateWords(blank) = %+v, want none", got)
	}
}
