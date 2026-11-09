package textutil

import (
	"reflect"
	"testing"
)

func TestWords(t *testing.T) {
	got := Words("  The\tcat\nsat  on the mat.  ")
	want := []string{"The", "cat", "sat", "on", "the", "mat."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Words() = %q, want %q", got, want)
	}
	if got := Words("   "); len(got) != 0 {
		t.Fatalf("Words(blank) = %q, want none", got)
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		word string
		want string
	}{
		{"Hello,", "hello"},
		{"WORLD!", "world"},
		{"don't", "don't"},
		{"it\u2019s", "it's"},
		{"`quoted`", "'quoted'"},
		{"Rock & Roll", "rockandroll"},
		{"chapter 12", "chapter12"},
		{"\u2014", ""},
		{"...", ""},
	}
	for _, test := range tests {
		if got := Normalize(test.word); got != test.want {
			t.Errorf("Normalize(%q) = %q, want %q", test.word, got, test.want)
		}
	}
}

func TestIsWord(t *testing.T) {
	tests := []struct {
		token string
		want  bool
	}{
		{"word", true},
		{"42", true},
		{"...", false},
		{"\u2014", false},
		{"", false},
	}
	for _, test := range tests {
		if got := IsWord(test.token); got != test.want {
			t.Errorf("IsWord(%q) = %v, want %v", test.token, got, test.want)
		}
	}
}

func TestNormalizeUnicodeAndTypography(t *testing.T) {
	tests := []struct {
		word string
		want string
	}{
		{"Caf\u00e9", "caf\u00e9"},
		{"\u201cQuoted\u201d", "quoted"},
		{"well\u2011known", "wellknown"},
		{"rock\u00a0star", "rock star"},
		{"can\u02bct", "can't"},
	}
	for _, test := range tests {
		if got := Normalize(test.word); got != test.want {
			t.Errorf("Normalize(%q) = %q, want %q", test.word, got, test.want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"cat", "cat", 0},
		{"cat", "cot", 1},
		{"kitten", "sitting", 3},
		{"caf\u00e9", "cafe", 1},
	}
	for _, test := range tests {
		if got := Levenshtein(test.a, test.b); got != test.want {
			t.Errorf("Levenshtein(%q, %q) = %d, want %d", test.a, test.b, got, test.want)
		}
	}
}
