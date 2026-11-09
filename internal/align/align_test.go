package align

import (
	"strings"
	"testing"
)

func words(text string) []string {
	return strings.Fields(text)
}

func TestSequencesIdentical(t *testing.T) {
	source := words("one two three four five six seven eight nine ten")
	result := Sequences(source, source)

	if result.Matched != len(source) || result.Total != len(source) {
		t.Fatalf("Matched = %d/%d, want %d/%d", result.Matched, result.Total, len(source), len(source))
	}
	if result.Confidence != 1 {
		t.Fatalf("Confidence = %v, want 1", result.Confidence)
	}
	for index, target := range result.Match {
		if target != index {
			t.Fatalf("Match[%d] = %d, want %d", index, target, index)
		}
	}
}

func TestSequencesTargetHasIntro(t *testing.T) {
	source := words("one two three four five six seven eight nine ten")
	target := words("intro one two three four five six seven eight nine ten")
	result := Sequences(source, target)

	for index := range source {
		if result.Match[index] != index+1 {
			t.Fatalf("Match[%d] = %d, want %d", index, result.Match[index], index+1)
		}
	}
	if result.Confidence != 1 {
		t.Fatalf("Confidence = %v, want 1", result.Confidence)
	}
}

func TestSequencesSourceHasExtraWord(t *testing.T) {
	source := words("one two noise three four five")
	target := words("one two three four five")
	result := Sequences(source, target)

	want := []int{0, 1, -1, 2, 3, 4}
	for index, targetIndex := range want {
		if result.Match[index] != targetIndex {
			t.Fatalf("Match[%d] = %d, want %d", index, result.Match[index], targetIndex)
		}
	}
	if result.Matched != 5 || result.Total != 6 {
		t.Fatalf("Matched = %d/%d, want 5/6", result.Matched, result.Total)
	}
	if result.Confidence != 5.0/6.0 {
		t.Fatalf("Confidence = %v, want %v", result.Confidence, 5.0/6.0)
	}
}

func TestSequencesSubstitution(t *testing.T) {
	source := words("one two three four")
	target := words("one too three four")
	result := Sequences(source, target)

	if result.Match[0] != 0 || result.Match[2] != 2 || result.Match[3] != 3 {
		t.Fatalf("Match = %v, want words around the substitution matched", result.Match)
	}
	if result.Match[1] != -1 {
		t.Fatalf("Match[1] = %d, want -1 for the substituted word", result.Match[1])
	}
	if result.Confidence != 0.75 {
		t.Fatalf("Confidence = %v, want 0.75", result.Confidence)
	}
}

func TestSequencesEmpty(t *testing.T) {
	for _, test := range []struct {
		name   string
		source []string
		target []string
	}{
		{"both empty", nil, nil},
		{"empty source", nil, words("one two three")},
		{"empty target", words("one two three"), nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := Sequences(test.source, test.target)
			if result.Total != len(test.source) || result.Matched != 0 {
				t.Fatalf("Matched = %d/%d, want 0/%d", result.Matched, result.Total, len(test.source))
			}
			for index, target := range result.Match {
				if target != -1 {
					t.Fatalf("Match[%d] = %d, want -1", index, target)
				}
			}
		})
	}
}

func TestSequencesShortInputs(t *testing.T) {
	// Shorter than the anchor size, so the dynamic programming path runs
	// without any anchor.
	result := Sequences(words("a b"), words("x a b"))
	if result.Match[0] != 1 || result.Match[1] != 2 {
		t.Fatalf("Match = %v, want [1 2]", result.Match)
	}
}
