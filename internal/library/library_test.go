package library

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mindsgn-studio/amba/internal/store"
)

func writeFile(t *testing.T, filename string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanGroupsAndOrders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Author", "Book One", "part 10.mp3"), "audio")
	writeFile(t, filepath.Join(root, "Author", "Book One", "part 2.mp3"), "audio")
	writeFile(t, filepath.Join(root, "Author", "Book One", "book.epub"), "not a real EPUB")
	writeFile(t, filepath.Join(root, "Book Two", "02.mp3"), "audio")
	writeFile(t, filepath.Join(root, "Book Two", "01.mp3"), "audio")
	writeFile(t, filepath.Join(root, "Book Two", "10.mp3"), "audio")
	writeFile(t, filepath.Join(root, "Unsupported", "story.m4a"), "audio")
	writeFile(t, filepath.Join(root, "Empty", "notes.txt"), "text")
	writeFile(t, filepath.Join(root, ".hidden", "secret.mp3"), "audio")

	books, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 {
		t.Fatalf("Scan() found %d books, want 2: %+v", len(books), books)
	}

	first := books[0]
	if first.Title != "Book One" {
		t.Errorf("first Title = %q, want %q", first.Title, "Book One")
	}
	wantAudio := []string{"part 2.mp3", "part 10.mp3"}
	if got := basenames(first.AudioFiles); !reflect.DeepEqual(got, wantAudio) {
		t.Errorf("first AudioFiles = %q, want %q", got, wantAudio)
	}
	if filepath.Base(first.EPUBPath) != "book.epub" {
		t.Errorf("first EPUBPath = %q, want the book beside the audio", first.EPUBPath)
	}

	second := books[1]
	if second.Title != "Book Two" {
		t.Errorf("second Title = %q, want %q", second.Title, "Book Two")
	}
	wantAudio = []string{"01.mp3", "02.mp3", "10.mp3"}
	if got := basenames(second.AudioFiles); !reflect.DeepEqual(got, wantAudio) {
		t.Errorf("second AudioFiles = %q, want %q", got, wantAudio)
	}
	if second.EPUBPath != "" {
		t.Errorf("second EPUBPath = %q, want none", second.EPUBPath)
	}

	if first.ID == "" || second.ID == "" || first.ID == second.ID {
		t.Errorf("IDs = %q and %q, want two distinct non-empty IDs", first.ID, second.ID)
	}

	// A second scan must give the same IDs, or progress would be lost.
	again, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != len(books) {
		t.Fatalf("second Scan() found %d books, want %d", len(again), len(books))
	}
	for index := range books {
		if again[index].ID != books[index].ID {
			t.Errorf("book %d ID changed from %q to %q", index, books[index].ID, again[index].ID)
		}
	}
}

func TestScanIgnoresUnsupportedAudio(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Book", "story.m4a"), "audio")

	books, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 0 {
		t.Fatalf("Scan() found %d books, want none: %+v", len(books), books)
	}
}

func TestScanRejectsBadRoot(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Scan() accepted a missing folder")
	}

	filename := filepath.Join(t.TempDir(), "file")
	writeFile(t, filename, "")
	if _, err := Scan(filename); err == nil {
		t.Fatal("Scan() accepted a file")
	}
}

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  bool
	}{
		{"2", "10", true},
		{"part 2", "part 10", true},
		{"Part 2", "part 10", true},
		{"10", "2", false},
		{"a", "B", true},
		{"same", "same", false},
	}
	for _, test := range tests {
		if got := NaturalLess(test.left, test.right); got != test.want {
			t.Errorf("NaturalLess(%q, %q) = %v, want %v", test.left, test.right, got, test.want)
		}
	}
}

func TestProgressStoreRoundTrip(t *testing.T) {
	cache, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	progress := NewProgressStore(cache)

	if got := progress.Load("book"); got != (Progress{}) {
		t.Fatalf("Load() = %+v, want the zero value", got)
	}

	saved := Progress{
		Position: 2 * time.Hour,
		Duration: 6 * time.Hour,
		Speed:    1.25,
	}
	if err := progress.Save("book", saved); err != nil {
		t.Fatal(err)
	}

	loaded := progress.Load("book")
	if loaded.Position != saved.Position || loaded.Duration != saved.Duration || loaded.Speed != saved.Speed {
		t.Fatalf("Load() = %+v, want %+v", loaded, saved)
	}
	if loaded.Updated.IsZero() {
		t.Error("Save() left Updated zero")
	}
}

func TestProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress Progress
		fraction float64
		started  bool
		finished bool
	}{
		{"unstarted", Progress{}, 0, false, false},
		{"half", Progress{Position: 3 * time.Hour, Duration: 6 * time.Hour}, 0.5, true, false},
		{"finished", Progress{Position: 6 * time.Hour, Duration: 6 * time.Hour}, 1, true, true},
		{"near end", Progress{Position: 6*time.Hour - 3*time.Second, Duration: 6 * time.Hour}, float64(6*time.Hour-3*time.Second) / float64(6*time.Hour), true, true},
		{"no duration", Progress{Position: time.Hour}, 0, true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.progress.Fraction(); got != test.fraction {
				t.Errorf("Fraction() = %v, want %v", got, test.fraction)
			}
			if got := test.progress.Started(); got != test.started {
				t.Errorf("Started() = %v, want %v", got, test.started)
			}
			if got := test.progress.Finished(); got != test.finished {
				t.Errorf("Finished() = %v, want %v", got, test.finished)
			}
		})
	}
}

func basenames(files []string) []string {
	names := make([]string, len(files))
	for index, file := range files {
		names[index] = filepath.Base(file)
	}
	return names
}
