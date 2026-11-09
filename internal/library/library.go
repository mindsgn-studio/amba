// Package library finds the audiobooks on disk and remembers how far the
// listener has got in each one.
package library

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/store"
)

// Book is one audiobook and the book that goes with it, if any.
type Book struct {
	ID             string
	Title          string
	Author         string
	Directory      string
	AudioFiles     []string
	EPUBPath       string
	Cover          []byte
	CoverMediaType string
}

// Progress is how far the listener has got in a book.
type Progress struct {
	Position time.Duration `json:"position"`
	Duration time.Duration `json:"duration"`
	Speed    float64       `json:"speed"`
	Updated  time.Time     `json:"updated"`
}

// Fraction is the share of the book heard, from 0 to 1.
func (p Progress) Fraction() float64 {
	if p.Duration <= 0 || p.Position <= 0 {
		return 0
	}
	fraction := float64(p.Position) / float64(p.Duration)
	if fraction > 1 {
		return 1
	}
	return fraction
}

// Started reports that the listener has heard part of the book.
func (p Progress) Started() bool { return p.Position > 0 }

// Finished reports that the listener has reached the end of the book.
func (p Progress) Finished() bool {
	return p.Duration > 0 && p.Position >= p.Duration-5*time.Second
}

// ProgressStore keeps one progress file per book in the app's cache.
type ProgressStore struct {
	cache *store.Store
	mu    sync.Mutex
}

// NewProgressStore creates a progress store in cache.
func NewProgressStore(cache *store.Store) *ProgressStore {
	return &ProgressStore{cache: cache}
}

// Load returns a book's progress, or the zero value when it has none.
func (p *ProgressStore) Load(bookID string) Progress {
	p.mu.Lock()
	defer p.mu.Unlock()

	var progress Progress
	_ = p.cache.ReadJSON(p.path(bookID), &progress)
	return progress
}

// Save writes a book's progress.
func (p *ProgressStore) Save(bookID string, progress Progress) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if progress.Updated.IsZero() {
		progress.Updated = time.Now()
	}
	return p.cache.WriteJSON(p.path(bookID), progress)
}

func (p *ProgressStore) path(bookID string) string {
	return p.cache.File("progress", bookID+".json")
}

// audioExtensions are the containers the player can decode. They must stay in
// step with the decoders registered in package player; a book in any other
// format would show up but fail to play.
var audioExtensions = map[string]bool{
	".mp3": true,
	".ogg": true,
	".wav": true,
}

// Scan finds every folder under root that holds audio files and makes it a
// book. Audio files in one folder are one book, in natural order, and an
// EPUB beside them supplies the text.
func Scan(root string) ([]Book, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("read audiobooks folder: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("audiobooks path is not a folder: %s", root)
	}

	type group struct {
		audio []string
		epub  []string
	}
	groups := make(map[string]*group)

	err = filepath.WalkDir(root, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if filename != root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			return nil
		}

		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if !audioExtensions[extension] && extension != ".epub" {
			return nil
		}

		directory := filepath.Dir(filename)
		item := groups[directory]
		if item == nil {
			item = &group{}
			groups[directory] = item
		}
		if extension == ".epub" {
			item.epub = append(item.epub, filename)
		} else {
			item.audio = append(item.audio, filename)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan audiobooks: %w", err)
	}

	books := make([]Book, 0, len(groups))
	for directory, item := range groups {
		if len(item.audio) == 0 {
			continue
		}

		sort.Slice(item.audio, func(i, j int) bool {
			return NaturalLess(filepath.Base(item.audio[i]), filepath.Base(item.audio[j]))
		})
		sort.Slice(item.epub, func(i, j int) bool {
			return NaturalLess(filepath.Base(item.epub[i]), filepath.Base(item.epub[j]))
		})

		relative, relErr := filepath.Rel(root, directory)
		if relErr != nil {
			relative = directory
		}

		book := Book{
			ID:         bookID(relative),
			Title:      humanize(filepath.Base(directory)),
			Directory:  directory,
			AudioFiles: item.audio,
		}

		if len(item.epub) > 0 {
			book.EPUBPath = item.epub[0]
			if metadata, metaErr := epub.MetadataOf(book.EPUBPath); metaErr == nil {
				if metadata.Title != "" {
					book.Title = metadata.Title
				}
				book.Author = metadata.Author
				book.Cover = metadata.Cover
				book.CoverMediaType = metadata.CoverMediaType
			}
		}

		books = append(books, book)
	}

	sort.Slice(books, func(i, j int) bool {
		if books[i].Title == books[j].Title {
			return books[i].ID < books[j].ID
		}
		return NaturalLess(books[i].Title, books[j].Title)
	})
	return books, nil
}

func bookID(relative string) string {
	sum := sha256.Sum256([]byte(filepath.ToSlash(relative)))
	return hex.EncodeToString(sum[:12])
}

// humanize turns a folder name into a title.
func humanize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(name)
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "Untitled"
	}

	words := strings.Fields(name)
	for index, word := range words {
		if word == strings.ToLower(word) {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[index] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// NaturalLess orders names as a person reads them: "2" before "10", and
// letters without regard to case.
func NaturalLess(a, b string) bool {
	return naturalCompare(a, b) < 0
}

func naturalCompare(a, b string) int {
	left := []rune(strings.ToLower(a))
	right := []rune(strings.ToLower(b))
	i, j := 0, 0

	for i < len(left) && j < len(right) {
		if unicode.IsDigit(left[i]) && unicode.IsDigit(right[j]) {
			leftStart, rightStart := i, j
			for i < len(left) && unicode.IsDigit(left[i]) {
				i++
			}
			for j < len(right) && unicode.IsDigit(right[j]) {
				j++
			}

			leftNumber := strings.TrimLeft(string(left[leftStart:i]), "0")
			rightNumber := strings.TrimLeft(string(right[rightStart:j]), "0")
			if len(leftNumber) != len(rightNumber) {
				if len(leftNumber) < len(rightNumber) {
					return -1
				}
				return 1
			}
			if leftNumber != rightNumber {
				if leftNumber < rightNumber {
					return -1
				}
				return 1
			}
			continue
		}

		if left[i] != right[j] {
			if left[i] < right[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}

	switch {
	case i < len(left):
		return 1
	case j < len(right):
		return -1
	default:
		return 0
	}
}
