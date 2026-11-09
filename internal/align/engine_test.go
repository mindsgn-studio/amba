package align

import (
	"context"
	"strings"
	"testing"
)

func TestWordSimilarity(t *testing.T) {
	tests := []struct {
		a, b string
		want float64
		op   string
	}{
		{"hello", "hello", 1, "=="},
		{"Hello,", "hello", 1, "=="},
		{"it\u2019s", "it's", 1, "=="},
		{"colour", "color", 0.833, ">="},
		{"cat", "cot", 0.666, ">="},
		{"cat", "elephant", 0.35, "<="},
		{"", "word", 0, "=="},
		{"a", "i", 0, "=="},
	}
	for _, test := range tests {
		got := WordSimilarity(test.a, test.b)
		switch test.op {
		case "==":
			if got != test.want {
				t.Errorf("WordSimilarity(%q, %q) = %v, want %v", test.a, test.b, got, test.want)
			}
		case ">=":
			if got < test.want {
				t.Errorf("WordSimilarity(%q, %q) = %v, want at least %v", test.a, test.b, got, test.want)
			}
		case "<=":
			if got > test.want {
				t.Errorf("WordSimilarity(%q, %q) = %v, want at most %v", test.a, test.b, got, test.want)
			}
		}
	}
}

func TestAlignSequencesFuzzyWhisperErrors(t *testing.T) {
	source := strings.Fields("the silver dragon guarded the ancient gate while travellers rested")
	target := strings.Fields("the silver drag0n guarded the ancyent gate while travelers rested")
	result := AlignSequences(source, target)

	if !result.Accepted {
		t.Fatalf("Accepted = false: %v", result.Diagnostics)
	}
	if result.EbookCoverage < 0.9 {
		t.Errorf("EbookCoverage = %v, want at least 0.9", result.EbookCoverage)
	}
	if result.Quality < 0.8 {
		t.Errorf("Quality = %v, want at least 0.8", result.Quality)
	}
	if len(result.Matches) < 9 {
		t.Errorf("Matches = %d, want at least 9", len(result.Matches))
	}
}

func TestAlignSequencesConfidenceAndDiagnostics(t *testing.T) {
	source := strings.Fields("alpha bravo charlie delta echo foxtrot")
	target := strings.Fields("zulu yankee xray whiskey victor uniform")

	result := AlignSequences(source, target)
	if result.Accepted {
		t.Fatalf("unrelated sequences were accepted: %+v", result)
	}
	if result.Confidence >= 0.8 {
		t.Errorf("Confidence = %v, want below 0.8", result.Confidence)
	}
	if len(result.Diagnostics) == 0 {
		t.Error("rejection did not explain itself")
	}
}

func TestAlignChaptersIndependent(t *testing.T) {
	source := strings.Fields(
		"alpha bravo charlie delta echo foxtrot " +
			"zulu yankee xray whiskey victor uniform " +
			"golf hotel india juliet kilo lima",
	)
	target := strings.Fields(
		"alpha bravo charlie delta echo foxtrot " +
			"something else entirely unrelated here now " +
			"golf hotel india juliet kilo lima",
	)
	ranges := []Range{{Start: 0, End: 6}, {Start: 6, End: 12}, {Start: 12, End: 18}}
	result := AlignChapters(source, target, ranges, nil, nil, DefaultConfig())

	if len(result.Chapters) != 3 {
		t.Fatalf("Chapters = %d, want 3", len(result.Chapters))
	}
	if !result.Chapters[0].Accepted || !result.Chapters[2].Accepted {
		t.Errorf("matching chapters were rejected: %+v", result.Chapters)
	}
	if result.Chapters[1].Accepted {
		t.Errorf("unrelated chapter was accepted: %v", result.Chapters[1].Diagnostics)
	}
	if result.Chapters[0].AudioStart != 0 || result.Chapters[2].AudioStart != 12 {
		t.Errorf("chapter audio ranges = %d and %d, want 0 and 12", result.Chapters[0].AudioStart, result.Chapters[2].AudioStart)
	}
}

func TestLocalAlignFindsPassage(t *testing.T) {
	source := strings.Fields("one two three four five six seven eight")
	target := strings.Fields("noise noise three four five six more noise")
	result := LocalAlign(source, target, DefaultConfig())

	if len(result.Matches) < 4 {
		t.Fatalf("LocalAlign found %d matches, want at least 4", len(result.Matches))
	}
	for _, match := range result.Matches {
		if match.EbookWordIndex < 2 || match.EbookWordIndex > 5 {
			t.Errorf("local match at ebook index %d, want the shared passage", match.EbookWordIndex)
		}
	}
}

func TestSequencesRemainsMonotonic(t *testing.T) {
	source := strings.Fields("one two three four five six seven eight nine ten")
	target := strings.Fields("zero one two three four five six seven eight nine ten eleven")
	result := Sequences(source, target)

	previous := -1
	for index, value := range result.Match {
		if value < 0 {
			continue
		}
		if value <= previous {
			t.Fatalf("Match[%d] = %d after %d, want monotonic", index, value, previous)
		}
		previous = value
	}
}

func TestAlignContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := strings.Fields("alpha bravo charlie delta echo")
	target := strings.Fields("alpha bravo charlie delta echo")
	if _, err := AlignSequencesContext(ctx, source, target, DefaultConfig(), nil); err == nil {
		t.Fatal("cancelled alignment returned no error")
	}
}

func TestAlignProgress(t *testing.T) {
	var fractions []float64
	source := strings.Fields("alpha bravo charlie delta echo foxtrot golf hotel")
	target := strings.Fields("alpha bravo charlie delta echo foxtrot golf hotel")
	result, err := AlignSequencesContext(context.Background(), source, target, DefaultConfig(), func(fraction float64) {
		fractions = append(fractions, fraction)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted {
		t.Fatalf("identical sequences were rejected: %v", result.Diagnostics)
	}
	if len(fractions) < 2 {
		t.Fatalf("progress = %v, want several updates", fractions)
	}
	if fractions[len(fractions)-1] != 1 {
		t.Errorf("last progress = %v, want 1", fractions[len(fractions)-1])
	}
}
