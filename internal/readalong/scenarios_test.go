package readalong

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mindsgn-studio/amba/internal/align"
	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

func scenarioWords(prefix string, count int) []string {
	words := make([]string, count)
	for index := range words {
		words[index] = fmt.Sprintf("%s%04d", prefix, index)
	}
	return words
}

func scenarioChapter(title string, words []string, perParagraph int) epub.Chapter {
	chapter := epub.Chapter{Title: title}
	for start := 0; start < len(words); start += perParagraph {
		end := start + perParagraph
		if end > len(words) {
			end = len(words)
		}
		chapter.Paragraphs = append(chapter.Paragraphs, epub.Paragraph{
			Text:  strings.Join(words[start:end], " "),
			Words: append([]string(nil), words[start:end]...),
		})
	}
	return chapter
}

func scenarioTranscript(words []string) *transcribe.Transcript {
	transcript := &transcribe.Transcript{
		Files:    []transcribe.File{{Path: "book.mp3", Duration: time.Duration(len(words)) * 500 * time.Millisecond}},
		Duration: time.Duration(len(words)) * 500 * time.Millisecond,
	}
	for index, word := range words {
		start := time.Duration(index) * 500 * time.Millisecond
		transcript.Words = append(transcript.Words, transcribe.Word{
			Text:  word,
			Start: start,
			End:   start + 400*time.Millisecond,
		})
	}
	return transcript
}

func scenarioBook(prefix string, chapters, perChapter int) ([]epub.Chapter, []string) {
	var book []epub.Chapter
	var words []string
	for index := 0; index < chapters; index++ {
		chapterWords := scenarioWords(fmt.Sprintf("%s%02d", prefix, index), perChapter)
		book = append(book, scenarioChapter(fmt.Sprintf("Chapter %d", index+1), chapterWords, 8))
		words = append(words, chapterWords...)
	}
	return book, words
}

func TestScenarioAIdenticalEditions(t *testing.T) {
	chapters, words := scenarioBook("alpha", 4, 30)
	content, err := FromEPUB(chapters, scenarioTranscript(words))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false, confidence %v, diagnostics %v", content.Confidence, content.Diagnostics)
	}
	if content.Confidence < 0.99 {
		t.Errorf("Confidence = %v, want at least 0.99", content.Confidence)
	}
	for index, chapter := range content.Chapters {
		if !chapter.Accepted {
			t.Errorf("chapter %d not accepted: %v", index, chapter.Diagnostics)
		}
	}
	if len(content.Alignment.Matches) != len(words) {
		t.Errorf("matched %d words, want %d", len(content.Alignment.Matches), len(words))
	}
}

func TestScenarioBWhisperErrors(t *testing.T) {
	chapters, words := scenarioBook("bravo", 4, 40)
	audio := append([]string(nil), words...)
	for index := 3; index < len(audio); index += 37 {
		word := audio[index]
		audio[index] = word[:len(word)-1] + "z"
	}
	content, err := FromEPUB(chapters, scenarioTranscript(audio))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false, confidence %v, diagnostics %v", content.Confidence, content.Diagnostics)
	}
	if content.Confidence < 0.85 {
		t.Errorf("Confidence = %v, want at least 0.85", content.Confidence)
	}
	if content.EbookCoverage < 0.9 {
		t.Errorf("EbookCoverage = %v, want fuzzy matching to recover most words", content.EbookCoverage)
	}
}

func TestScenarioCDifferentEditions(t *testing.T) {
	chapters, words := scenarioBook("charlie", 4, 40)
	audio := append([]string(nil), words...)
	// Rewrite the third chapter almost completely.
	for index := 2 * 40; index < 3*40; index++ {
		audio[index] = fmt.Sprintf("different%04d", index)
	}
	content, err := FromEPUB(chapters, scenarioTranscript(audio))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false, want accepted chapters around the rewritten one")
	}
	if content.Chapters[2].Accepted {
		t.Errorf("rewritten chapter was accepted: %v", content.Chapters[2].Diagnostics)
	}
	for _, index := range []int{0, 1, 3} {
		if !content.Chapters[index].Accepted {
			t.Errorf("chapter %d not accepted: %v", index, content.Chapters[index].Diagnostics)
		}
	}
	if content.Chapters[2].Source != "transcript" {
		t.Errorf("rejected chapter source = %q, want transcript fallback", content.Chapters[2].Source)
	}
}

func TestScenarioDMissingParagraphs(t *testing.T) {
	chapters, words := scenarioBook("delta", 4, 40)
	var audio []string
	audio = append(audio, words[:70]...)
	audio = append(audio, words[80:]...)

	content, err := FromEPUB(chapters, scenarioTranscript(audio))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false, confidence %v, diagnostics %v", content.Confidence, content.Diagnostics)
	}
	// The words after the omitted paragraph must still point at the right
	// narration positions, not be shifted by the missing words.
	for _, match := range content.Alignment.Matches {
		if match.EbookWordIndex >= 80 {
			want := match.EbookWordIndex - 10
			if match.AudioWordIndex != want {
				t.Fatalf("ebook word %d mapped to audio %d, want %d", match.EbookWordIndex, match.AudioWordIndex, want)
			}
		}
	}
}

func TestScenarioEAdditionalAudio(t *testing.T) {
	chapters, words := scenarioBook("echo", 3, 30)
	audio := append(scenarioWords("intro", 25), words...)
	audio = append(audio, scenarioWords("outro", 20)...)

	content, err := FromEPUB(chapters, scenarioTranscript(audio))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false, confidence %v, diagnostics %v", content.Confidence, content.Diagnostics)
	}
	for index, chapter := range content.Chapters {
		if !chapter.Accepted {
			t.Errorf("chapter %d not accepted with extra audio: %v", index, chapter.Diagnostics)
		}
	}
	if content.AudioCoverage >= 0.99 {
		t.Errorf("AudioCoverage = %v, want extra audio to count as unmatched", content.AudioCoverage)
	}
}

func TestScenarioFUnrelatedBooks(t *testing.T) {
	chapters, _ := scenarioBook("foxtrot", 3, 40)
	audio := scenarioWords("zulu", 120)

	content, err := FromEPUB(chapters, scenarioTranscript(audio))
	if err != nil {
		t.Fatal(err)
	}
	if content.Synced {
		t.Fatalf("Synced = true for unrelated books, confidence %v", content.Confidence)
	}
	if content.Confidence >= 0.80 {
		t.Errorf("Confidence = %v, want below 0.80", content.Confidence)
	}
}

func TestScenarioGPartiallyMatchingChapters(t *testing.T) {
	chapters, words := scenarioBook("golf", 4, 35)
	audio := append([]string(nil), words...)
	for index := 35; index < 70; index++ {
		audio[index] = fmt.Sprintf("unrelated%04d", index)
	}
	content, err := FromEPUB(chapters, scenarioTranscript(audio))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatal("Synced = false, want the matching chapters to remain usable")
	}
	if !content.Chapters[0].Accepted || !content.Chapters[2].Accepted || !content.Chapters[3].Accepted {
		t.Errorf("matching chapters not accepted: %+v", content.Chapters)
	}
	if content.Chapters[1].Accepted {
		t.Errorf("unrelated chapter accepted: %v", content.Chapters[1].Diagnostics)
	}
}

func TestScenarioHDifferentChapterBoundaries(t *testing.T) {
	chapters, words := scenarioBook("hotel", 5, 25)
	// The narration has file boundaries that do not match the ebook's
	// chapter boundaries. Alignment must use passage content, not numbers.
	transcript := scenarioTranscript(words)
	transcript.Files = []transcribe.File{
		{Path: "one.mp3", Start: 0, Duration: 7 * time.Second},
		{Path: "two.mp3", Start: 7 * time.Second, Duration: 11 * time.Second},
		{Path: "three.mp3", Start: 18 * time.Second, Duration: 40 * time.Second},
	}
	content, err := FromEPUB(chapters, transcript)
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false, confidence %v, diagnostics %v", content.Confidence, content.Diagnostics)
	}
	for index, chapter := range content.Chapters {
		if !chapter.Accepted {
			t.Errorf("chapter %d not accepted: %v", index, chapter.Diagnostics)
		}
	}
}

func TestScenarioIRepeatedPhrases(t *testing.T) {
	repeated := strings.Fields("the quick brown fox jumps over the lazy dog again and again")
	var words []string
	for index := 0; index < 8; index++ {
		words = append(words, repeated...)
		words = append(words, scenarioWords(fmt.Sprintf("unique%d", index), 12)...)
	}
	var chapters []epub.Chapter
	for start := 0; start < len(words); start += 50 {
		end := start + 50
		if end > len(words) {
			end = len(words)
		}
		chapters = append(chapters, scenarioChapter(fmt.Sprintf("Chapter %d", len(chapters)+1), words[start:end], 10))
	}

	content, err := FromEPUB(chapters, scenarioTranscript(words))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false with repeated phrases, confidence %v, diagnostics %v", content.Confidence, content.Diagnostics)
	}
	// The alignment must remain monotonic even when phrases repeat.
	previous := -1
	for _, match := range content.Alignment.Matches {
		if match.AudioWordIndex <= previous {
			t.Fatalf("alignment went backwards at ebook word %d", match.EbookWordIndex)
		}
		previous = match.AudioWordIndex
	}
}

func TestScenarioJInvalidEPUB(t *testing.T) {
	transcript := scenarioTranscript(scenarioWords("juliet", 10))
	content, err := FromEPUB(nil, transcript)
	if err == nil {
		t.Fatal("FromEPUB() accepted no chapters")
	}
	if content != nil {
		t.Fatal("FromEPUB() returned content for an invalid book")
	}

	// The transcript-only path must keep working.
	fallback := FromTranscript(transcript)
	if fallback == nil || len(fallback.Chapters) == 0 {
		t.Fatal("FromTranscript() failed for a valid transcript")
	}
}

func TestConfidenceThresholdsAreConfigurable(t *testing.T) {
	chapters, words := scenarioBook("kilo", 1, 30)
	audio := append([]string(nil), words...)
	for index := 0; index < 8; index++ {
		audio[index] = fmt.Sprintf("noise%04d", index)
	}

	strict := align.DefaultConfig()
	strict.MinEbookCoverage = 0.95
	content, err := FromEPUBConfig(chapters, scenarioTranscript(audio), strict)
	if err != nil {
		t.Fatal(err)
	}
	if content.Chapters[0].Accepted {
		t.Error("strict configuration accepted a chapter below its coverage threshold")
	}

	loose := align.DefaultConfig()
	loose.MinConfidence = 0.5
	loose.MinEbookCoverage = 0.5
	loose.MinQuality = 0.5
	content, err = FromEPUBConfig(chapters, scenarioTranscript(audio), loose)
	if err != nil {
		t.Fatal(err)
	}
	if !content.Chapters[0].Accepted {
		t.Errorf("loose configuration rejected a chapter: %v", content.Chapters[0].Diagnostics)
	}
}

func TestUnmatchedWordsNeverGetAnInventedTimestamp(t *testing.T) {
	words := []string{"alpha", "bravo", "charlie", "delta", "missing", "echo", "foxtrot", "golf", "hotel", "india"}
	audio := append([]string(nil), words[:4]...)
	audio = append(audio, words[5:]...)
	chapters := []epub.Chapter{scenarioChapter("Chapter", words, 10)}
	content, err := FromEPUB(chapters, scenarioTranscript(audio))
	if err != nil {
		t.Fatal(err)
	}
	if !content.Synced {
		t.Fatalf("Synced = false: %v", content.Diagnostics)
	}

	var found bool
	for _, paragraph := range content.Chapters[0].Paragraphs {
		for _, token := range paragraph.Tokens {
			if token.Normalized != "missing" {
				continue
			}
			found = true
			if token.Audio != -1 {
				t.Errorf("unmatched word %q has audio index %d, want -1", token.Text, token.Audio)
			}
		}
	}
	if !found {
		t.Fatal("unmatched word was not displayed")
	}
}

func TestAlignmentKeyChangesWithInputsAndConfig(t *testing.T) {
	chapters, words := scenarioBook("mike", 2, 20)
	transcript := scenarioTranscript(words)
	config := align.DefaultConfig()

	key := AlignmentKey(chapters, transcript, config)
	if key == "" || key != AlignmentKey(chapters, transcript, config) {
		t.Fatal("alignment key is not stable for unchanged inputs")
	}

	changedBook := append([]epub.Chapter(nil), chapters...)
	changedBook[0].Paragraphs = append([]epub.Paragraph(nil), chapters[0].Paragraphs...)
	changedBook[0].Paragraphs[0].Words = append([]string(nil), chapters[0].Paragraphs[0].Words...)
	changedBook[0].Paragraphs[0].Words[0] = "edited"
	if AlignmentKey(changedBook, transcript, config) == key {
		t.Error("changed ebook text did not invalidate the key")
	}

	changedTranscript := scenarioTranscript(words)
	changedTranscript.Words[0].Text = "edited"
	if AlignmentKey(chapters, changedTranscript, config) == key {
		t.Error("changed transcript did not invalidate the key")
	}

	changedConfig := config
	changedConfig.MinConfidence = 0.9
	if AlignmentKey(chapters, transcript, changedConfig) == key {
		t.Error("changed configuration did not invalidate the key")
	}
}
