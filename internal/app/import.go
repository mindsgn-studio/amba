package app

import (
	"fmt"
	"os"
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

// applyImports attaches EPUBs the listener chose from outside the audiobook
// folder, so an imported book is still there after a restart.
func (a *App) applyImports(books []library.Book) []library.Book {
	for index := range books {
		if books[index].EPUBPath != "" {
			continue
		}
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
		a.openBook(updated)
	}
	return nil
}
