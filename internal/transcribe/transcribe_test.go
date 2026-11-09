package transcribe

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mindsgn-studio/amba/internal/store"
)

func TestTokensToWords(t *testing.T) {
	tokens := []string{
		"<|startoftranscript|>",
		"\u2581Hello",
		"\u2581world",
		".",
		"\u0120again",
	}
	timestamps := []float32{0, 0.2, 0.5, 0.7, 1.0}
	durations := []float32{0.1, 0.1, 0.2, 0.1, 0.3}
	offset := 10 * time.Second

	words := tokensToWords(tokens, timestamps, durations, offset)
	want := []Word{
		{Text: "Hello", Start: 10200 * time.Millisecond, End: 10300 * time.Millisecond},
		{Text: "world.", Start: 10500 * time.Millisecond, End: 10800 * time.Millisecond},
		{Text: "again", Start: 11 * time.Second, End: 11300 * time.Millisecond},
	}
	if len(words) != len(want) {
		t.Fatalf("tokensToWords() = %+v, want %+v", words, want)
	}
	for index, word := range words {
		if word.Text != want[index].Text || !near(word.Start, want[index].Start) || !near(word.End, want[index].End) {
			t.Errorf("word %d = %+v, want %+v", index, word, want[index])
		}
	}
}

// near compares times made from the float32 timestamps sherpa returns, which
// cannot represent a tenth of a second exactly.
func near(got, want time.Duration) bool {
	delta := got - want
	if delta < 0 {
		delta = -delta
	}
	return delta < time.Millisecond
}

func TestTokensToWordsUsesNextTimestampWhenDurationIsMissing(t *testing.T) {
	tokens := []string{"\u2581one", "\u2581two"}
	timestamps := []float32{0.5, 0.9}
	words := tokensToWords(tokens, timestamps, nil, 0)

	if len(words) != 2 {
		t.Fatalf("tokensToWords() = %+v, want two words", words)
	}
	if !near(words[0].End, 900*time.Millisecond) || !near(words[0].Start, 500*time.Millisecond) {
		t.Errorf("first word = %+v, want 0.5s to 0.9s", words[0])
	}
}

func TestTokensToWordsIgnoresEmptyInput(t *testing.T) {
	if got := tokensToWords(nil, nil, nil, time.Second); len(got) != 0 {
		t.Errorf("tokensToWords(nil) = %+v, want none", got)
	}
	if got := tokensToWords([]string{"<|endoftext|>"}, []float32{0}, nil, 0); len(got) != 0 {
		t.Errorf("tokensToWords(special tokens) = %+v, want none", got)
	}
}

func testService(t *testing.T) *Service {
	t.Helper()

	cache, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return New(Config{}, cache)
}

func TestTranscriptCache(t *testing.T) {
	service := testService(t)
	directory := t.TempDir()
	first := filepath.Join(directory, "one.mp3")
	second := filepath.Join(directory, "two.mp3")
	for _, filename := range []string{first, second} {
		if err := os.WriteFile(filename, []byte("audio"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatal(err)
	}

	transcript := &Transcript{
		Version: transcriptVersion,
		BookID:  "book",
		Files: []File{
			{Path: first, Size: firstInfo.Size(), ModTime: firstInfo.ModTime().UnixNano(), Duration: time.Minute},
			{Path: second, Size: secondInfo.Size(), ModTime: secondInfo.ModTime().UnixNano(), Duration: time.Minute},
		},
		Words:    []Word{{Text: "word", Start: 0, End: time.Second}},
		Duration: 2 * time.Minute,
	}
	if err := service.saveCache(transcript); err != nil {
		t.Fatal(err)
	}

	cached, ok := service.loadCached("book", []string{first, second})
	if !ok {
		t.Fatal("loadCached() missed a valid transcript")
	}
	if cached.BookID != "book" || len(cached.Words) != 1 {
		t.Fatalf("loadCached() = %+v, want the saved transcript", cached)
	}

	if _, ok := service.loadCached("other", []string{first, second}); ok {
		t.Error("loadCached() found a transcript for another book")
	}
	if _, ok := service.loadCached("book", []string{first}); ok {
		t.Error("loadCached() accepted a changed set of audio files")
	}

	// A changed audio file invalidates the transcript.
	if err := os.WriteFile(first, []byte("longer audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := service.loadCached("book", []string{first, second}); ok {
		t.Error("loadCached() accepted a changed audio file")
	}
}

func TestTranscriptCacheRejectsOldVersion(t *testing.T) {
	service := testService(t)
	filename := filepath.Join(t.TempDir(), "one.mp3")
	if err := os.WriteFile(filename, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}

	transcript := &Transcript{
		Version: transcriptVersion - 1,
		BookID:  "book",
		Files:   []File{{Path: filename, Size: 5, ModTime: 1}},
		Words:   []Word{{Text: "word"}},
	}
	if err := service.saveCache(transcript); err != nil {
		t.Fatal(err)
	}
	if _, ok := service.loadCached("book", []string{filename}); ok {
		t.Error("loadCached() accepted an old transcript version")
	}
}

func TestEnsureModelSkipsDownloadWhenInstalled(t *testing.T) {
	directory := t.TempDir()
	modelDir := filepath.Join(directory, "model")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"tiny.en-encoder.int8.onnx",
		"tiny.en-decoder.int8.onnx",
		"tiny.en-tokens.txt",
	} {
		if err := os.WriteFile(filepath.Join(modelDir, name), []byte("model"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	service := testService(t)
	service.modelDir = modelDir
	service.modelsDir = filepath.Join(directory, "models")
	service.modelURL = "file:///does/not/exist"

	var stages []Progress
	if err := service.EnsureModel(context.Background(), func(p Progress) { stages = append(stages, p) }); err != nil {
		t.Fatal(err)
	}
	if len(stages) != 1 || stages[0].Stage != "Speech model is ready." {
		t.Errorf("EnsureModel() reported %+v, want the ready stage only", stages)
	}
	if _, err := os.Stat(service.modelsDir); !os.IsNotExist(err) {
		t.Error("EnsureModel() started a download even though the model is installed")
	}
}
