package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/library"
)

type epubImport struct {
	Path     string    `json:"path"`
	Imported time.Time `json:"imported"`
}

func (a *App) importPath(bookID string) string {
	return a.cache.File("books", bookID, "epub-import.json")
}

// applyAttachments attaches EPUBs the listener chose from outside the
// audiobook folder, so an imported book is still there after a restart, and
// honours the EPUBs the listener removed. An explicit import takes precedence
// over an EPUB found beside the audiobook.
func (a *App) applyAttachments(books []library.Book) []library.Book {
	for index := range books {
		a.applyRemoval(&books[index])
		var record epubImport
		if err := a.cache.ReadJSON(a.importPath(books[index].ID), &record); err != nil || record.Path == "" {
			continue
		}
		if _, err := os.Stat(record.Path); err != nil {
			continue
		}
		books[index].EPUBPath = record.Path
		if metadata, err := epub.MetadataOf(record.Path); err == nil {
			if metadata.Title != "" {
				books[index].Title = metadata.Title
			}
			books[index].Author = metadata.Author
			books[index].Cover = metadata.Cover
			books[index].CoverMediaType = metadata.CoverMediaType
		}
	}
	return books
}

// importEPUB asks for an EPUB and attaches it to an audiobook. The audiobook
// transcript is reused; importing a book never starts a new transcription.
func (a *App) importEPUB(bookID string) {
	book, ok := a.bookByID(bookID)
	if !ok {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.window,
			Title:  "Import an EPUB for " + book.Title,
			Filters: []mygo.FileFilter{{
				Name:       "EPUB books",
				Extensions: []string{"epub"},
			}},
		})
		if err != nil {
			a.Update(func() {
				if a.reader.bookID == bookID {
					a.reader.importErr = err
				}
			})
			return
		}
		if len(paths) == 0 {
			return
		}
		a.Update(func() {
			if err := a.attachEPUB(bookID, paths[0]); err != nil && a.reader.bookID == bookID {
				a.reader.importErr = err
			}
		})
	}()
}

func (a *App) attachEPUB(bookID, filename string) error {
	metadata, err := epub.MetadataOf(filename)
	if err != nil {
		return fmt.Errorf("read the EPUB: %w", err)
	}

	bookIndex := -1
	for index := range a.books {
		if a.books[index].ID == bookID {
			bookIndex = index
			break
		}
	}
	if bookIndex < 0 {
		return fmt.Errorf("the audiobook is no longer in the library")
	}

	record := epubImport{Path: filename, Imported: time.Now()}
	if err := a.cache.WriteJSON(a.importPath(bookID), record); err != nil {
		return fmt.Errorf("remember the EPUB: %w", err)
	}
	// Attaching a book is an explicit choice, so it supersedes an earlier
	// removal of the same file.
	_ = a.cache.Remove(a.removalPath(bookID))

	a.books[bookIndex].EPUBPath = filename
	if metadata.Title != "" {
		a.books[bookIndex].Title = metadata.Title
	}
	a.books[bookIndex].Author = metadata.Author
	a.books[bookIndex].Cover = metadata.Cover
	a.books[bookIndex].CoverMediaType = metadata.CoverMediaType
	updated := a.books[bookIndex]

	if len(metadata.Cover) > 0 {
		if image, err := ui.DecodeImage(metadata.Cover); err == nil {
			a.covers[bookID] = image
		}
	}
	a.progressCache[bookID] = a.progress.Load(bookID)

	if a.reader.bookID == bookID {
		a.reader.importErr = nil
		a.reader.removeErr = nil
		a.openBook(updated)
	}
	return nil
}

// epubRemoval records that the listener detached a book's own EPUB. The file
// stays on disk; the record only keeps the scanner from attaching it again.
type epubRemoval struct {
	Path    string    `json:"path"`
	Removed time.Time `json:"removed"`
}

func (a *App) removalPath(bookID string) string {
	return a.cache.File("books", bookID, "epub-removed.json")
}

// applyRemoval clears an EPUB that the listener removed from a book. It matches
// the exact file, so a different EPUB added to the folder later is still
// picked up.
func (a *App) applyRemoval(book *library.Book) {
	if book.EPUBPath == "" {
		return
	}
	var record epubRemoval
	if err := a.cache.ReadJSON(a.removalPath(book.ID), &record); err != nil || record.Path == "" {
		return
	}
	if record.Path != book.EPUBPath {
		return
	}
	resetEPUBMetadata(book)
}

// resetEPUBMetadata returns a book to what its audiobook folder says about it,
// forgetting the removed ebook's title, author and cover.
func resetEPUBMetadata(book *library.Book) {
	book.EPUBPath = ""
	book.Title = library.TitleFor(book.Directory)
	book.Author = ""
	book.Cover = nil
	book.CoverMediaType = ""
}

// removeEPUB detaches the book's EPUB without touching the audiobook or the
// listener's place in it. An imported EPUB is forgotten; an EPUB that lives
// beside the audiobook is remembered as removed, because the app never deletes
// the listener's files.
func (a *App) removeEPUB(bookID string) error {
	bookIndex := -1
	for index := range a.books {
		if a.books[index].ID == bookID {
			bookIndex = index
			break
		}
	}
	if bookIndex < 0 {
		return fmt.Errorf("the audiobook is no longer in the library")
	}

	book := &a.books[bookIndex]
	if book.EPUBPath == "" {
		return nil
	}

	if err := a.cache.Remove(a.importPath(bookID)); err != nil {
		return fmt.Errorf("forget the imported EPUB: %w", err)
	}
	if insideDirectory(book.EPUBPath, book.Directory) {
		record := epubRemoval{Path: book.EPUBPath, Removed: time.Now()}
		if err := a.cache.WriteJSON(a.removalPath(bookID), record); err != nil {
			return fmt.Errorf("remember the removed EPUB: %w", err)
		}
	}

	resetEPUBMetadata(book)
	updated := *book
	delete(a.covers, bookID)

	if a.reader.bookID == bookID {
		a.reader.book = updated
		a.reader.importErr = nil
		a.reader.removeErr = nil
		a.detachEPUB()
	}
	return nil
}

// insideDirectory reports whether path is inside directory.
func insideDirectory(path, directory string) bool {
	relative, err := filepath.Rel(directory, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
