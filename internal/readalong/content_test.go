package readalong

import (
	"testing"
	"time"

	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

func spoken(text string, start time.Duration) transcribe.Word {
	return transcribe.Word{Text: text, Start: start, End: start + 400*time.Millisecond}
}

func testTranscript(words []transcribe.Word) *transcribe.Transcript {
	return &transcribe.Transcript{
		Words:    words,
		Duration: 4*time.Second + 500*time.Millisecond,
	}
}

func TestFromEPUBSyncsWords(t *testing.T) {
	chapters := []epub.Chapter{
		{
			Title: "One",
			Paragraphs: []epub.Paragraph{
				{Words: []string{"The", "cat", "sat", "on", "the", "mat."}},
			},
		},
		{
			Title: "Two",
			Paragraphs: []epub.Paragraph{
				{Words: []string{"Then", "it", "slept."}},
			},
		},
	}

	transcript := testTranscript([]transcribe.Word{
		spoken("The", 0),
		spoken("cat", 500*time.Millisecond),
		spoken("sat", time.Second),
		spoken("on", 1500*time.Millisecond),
		spoken("the", 2*time.Second),
		spoken("mat.", 2500*time.Millisecond),
		spoken("Then", 3*time.Second),
		spoken("it", 3500*time.Millisecond),
		spoken("slept.", 4*time.Second),
	})

	content, err := FromEPUB(chapters, transcript)
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Errorf("Synced = false, want true (confidence %v)", content.Confidence)
	}
	if content.Confidence != 1 {
		t.Errorf("Confidence = %v, want 1", content.Confidence)
	}
	if len(content.Chapters) != 2 || len(content.Flat) != 9 {
		t.Fatalf("content has %d chapters and %d words, want 2 and 9", len(content.Chapters), len(content.Flat))
	}

	first, _ := content.TokenAt(0)
	if first.Text != "The" || first.Normalized != "the" || first.Audio != 0 {
		t.Errorf("first token = %+v, want The at audio 0", first)
	}
	last, _ := content.TokenAt(8)
	if last.Text != "slept." || last.Audio != 8 {
		t.Errorf("last token = %+v, want slept. at audio 8", last)
	}

	tests := []struct {
		name     string
		position time.Duration
		audio    int
		token    int
		chapter  int
	}{
		{"before the narration", -time.Second, -1, -1, 0},
		{"first word", 0, 0, 0, 0},
		{"middle of the first chapter", 2600 * time.Millisecond, 5, 5, 0},
		{"second chapter", 3500 * time.Millisecond, 7, 7, 1},
		{"past the end", time.Minute, 8, 8, 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := content.ActiveAudio(test.position); got != test.audio {
				t.Errorf("ActiveAudio(%v) = %d, want %d", test.position, got, test.audio)
			}
			if got := content.ActiveToken(test.position); got != test.token {
				t.Errorf("ActiveToken(%v) = %d, want %d", test.position, got, test.token)
			}
			if got := content.ChapterAt(test.position); got != test.chapter {
				t.Errorf("ChapterAt(%v) = %d, want %d", test.position, got, test.chapter)
			}
		})
	}

	if got := content.ChapterStart(1); got != 3*time.Second {
		t.Errorf("ChapterStart(1) = %v, want 3s", got)
	}
	if got := content.ChapterStart(0); got != 0 {
		t.Errorf("ChapterStart(0) = %v, want 0", got)
	}
	if got := content.ChapterStart(99); got != 0 {
		t.Errorf("ChapterStart(99) = %v, want 0", got)
	}

	if _, ok := content.TokenAt(-1); ok {
		t.Error("TokenAt(-1) reported a word")
	}
	if _, ok := content.TokenAt(99); ok {
		t.Error("TokenAt(99) reported a word")
	}
}

func TestFromEPUBFallsBackWhenConfidenceIsLow(t *testing.T) {
	paragraph := epub.Paragraph{}
	for index := 0; index < 40; index++ {
		paragraph.Words = append(paragraph.Words, "word"+string(rune('a'+index)))
	}
	chapters := []epub.Chapter{{Title: "Different Edition", Paragraphs: []epub.Paragraph{paragraph}}}

	words := make([]transcribe.Word, 0, 20)
	for index := 0; index < 5; index++ {
		words = append(words, spoken("word"+string(rune('a'+index)), time.Duration(index)*time.Second))
	}
	for index := 0; index < 15; index++ {
		words = append(words, spoken("nonsense", time.Duration(5+index)*time.Second))
	}

	content, err := FromEPUB(chapters, testTranscript(words))
	if err != nil {
		t.Fatal(err)
	}
	if content.Synced {
		t.Errorf("Synced = true at confidence %v, want false", content.Confidence)
	}
	if content.Confidence >= MinSyncConfidence {
		t.Errorf("Confidence = %v, want below %v", content.Confidence, MinSyncConfidence)
	}
}

func TestFromEPUBRejectsEmptyInput(t *testing.T) {
	chapters := []epub.Chapter{{Title: "One", Paragraphs: []epub.Paragraph{{Words: []string{"word"}}}}}
	transcript := testTranscript([]transcribe.Word{spoken("word", 0)})

	if _, err := FromEPUB(nil, transcript); err == nil {
		t.Error("FromEPUB() accepted no chapters")
	}
	if _, err := FromEPUB(chapters, nil); err == nil {
		t.Error("FromEPUB() accepted no transcript")
	}
	if _, err := FromEPUB(chapters, &transcribe.Transcript{}); err == nil {
		t.Error("FromEPUB() accepted an empty transcript")
	}
	if _, err := FromEPUB([]epub.Chapter{{Title: "Empty"}}, transcript); err == nil {
		t.Error("FromEPUB() accepted chapters with no words")
	}
}

func TestFromTranscript(t *testing.T) {
	transcript := &transcribe.Transcript{
		Files: []transcribe.File{
			{Path: "part1.mp3", Start: 0, Duration: 5 * time.Second},
			{Path: "part2.mp3", Start: 5 * time.Second, Duration: 5 * time.Second},
		},
		Words: []transcribe.Word{
			spoken("a", 100*time.Millisecond),
			spoken("b", 600*time.Millisecond),
			spoken("c", 1100*time.Millisecond),
			spoken("d", 2500*time.Millisecond),
			spoken("e", 5100*time.Millisecond),
			spoken("f", 5600*time.Millisecond),
		},
		Duration: 6 * time.Second,
	}

	content := FromTranscript(transcript)
	if content.Synced {
		t.Error("Synced = true, want false for transcript-only content")
	}
	if len(content.Chapters) != 2 {
		t.Fatalf("content has %d chapters, want 2", len(content.Chapters))
	}
	if content.Chapters[0].Title != "Part 1" || content.Chapters[1].Title != "Part 2" {
		t.Errorf("chapter titles = %q and %q, want Part 1 and Part 2", content.Chapters[0].Title, content.Chapters[1].Title)
	}

	// The 1.4 second pause before "d" starts a new paragraph.
	if got := len(content.Chapters[0].Paragraphs); got != 2 {
		t.Fatalf("first chapter has %d paragraphs, want 2", got)
	}
	if got := len(content.Chapters[0].Paragraphs[0].Tokens); got != 3 {
		t.Errorf("first paragraph has %d words, want 3", got)
	}
	if got := len(content.Chapters[1].Paragraphs); got != 1 {
		t.Errorf("second chapter has %d paragraphs, want 1", got)
	}
	if len(content.Flat) != 6 {
		t.Fatalf("content has %d words, want 6", len(content.Flat))
	}

	if got := content.ChapterAt(5500 * time.Millisecond); got != 1 {
		t.Errorf("ChapterAt(5.5s) = %d, want 1", got)
	}
	if got := content.ActiveToken(700 * time.Millisecond); got != 1 {
		t.Errorf("ActiveToken(0.7s) = %d, want 1", got)
	}
	if got := content.ChapterStart(1); got != 5100*time.Millisecond {
		t.Errorf("ChapterStart(1) = %v, want 5.1s", got)
	}
}

func TestFromTranscriptEmpty(t *testing.T) {
	content := FromTranscript(&transcribe.Transcript{})
	if content == nil {
		t.Fatal("FromTranscript() = nil")
	}
	if len(content.Chapters) != 0 || len(content.Audio) != 0 {
		t.Fatalf("FromTranscript() = %+v, want empty content", content)
	}
	if got := content.ActiveToken(time.Second); got != -1 {
		t.Errorf("ActiveToken() = %d, want -1", got)
	}
}
