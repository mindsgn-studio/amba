// Package app is the read-along application: its screens, its navigation and
// the background work that prepares a book for reading.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/mindsgn-studio/amba/internal/library"
	"github.com/mindsgn-studio/amba/internal/store"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

// DefaultModelURL is the official sherpa-onnx Whisper tiny.en archive.
const DefaultModelURL = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-whisper-tiny.en.tar.bz2"

// Config tells the app where its books, models and cache live.
type Config struct {
	LibraryRoot string
	CacheRoot   string
	ModelDir    string
	ModelsDir   string
	ModelURL    string
}

// DefaultConfig finds the project's folders and the user's cache folder.
func DefaultConfig() Config {
	root := projectRoot()
	cache, err := os.UserCacheDir()
	if err != nil || cache == "" {
		cache = filepath.Join(root, ".cache")
	}
	return Config{
		LibraryRoot: filepath.Join(root, "audiobooks"),
		CacheRoot:   filepath.Join(cache, "Amba"),
		ModelDir:    filepath.Join(root, "models", "sherpa-onnx-whisper-tiny.en"),
		ModelsDir:   filepath.Join(root, "models"),
		ModelURL:    DefaultModelURL,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	if c.LibraryRoot == "" {
		c.LibraryRoot = defaults.LibraryRoot
	}
	if c.CacheRoot == "" {
		c.CacheRoot = defaults.CacheRoot
	}
	if c.ModelDir == "" {
		c.ModelDir = defaults.ModelDir
	}
	if c.ModelsDir == "" {
		c.ModelsDir = defaults.ModelsDir
	}
	if c.ModelURL == "" {
		c.ModelURL = defaults.ModelURL
	}
	return c
}

// App holds the window's state. Its fields are read and changed on the main
// thread; background work publishes results through Update.
type App struct {
	config      Config
	cache       *store.Store
	progress    *library.ProgressStore
	transcriber *transcribe.Service
	router      *ui.Router
	window      *mygo.Window

	books         []library.Book
	covers        map[string]ui.ImageSource
	booksErr      error
	scanning      bool
	scanStage     string
	progressCache map[string]library.Progress
	libraryList   ui.ListState
	libraryChoice int

	reader readerState
}

// New creates the application and its services.
func New(config Config) (*App, error) {
	config = config.withDefaults()

	cache, err := store.New(config.CacheRoot)
	if err != nil {
		return nil, fmt.Errorf("create app cache: %w", err)
	}

	application := &App{
		config:   config,
		cache:    cache,
		progress: library.NewProgressStore(cache),
		transcriber: transcribe.New(transcribe.Config{
			ModelDir:  config.ModelDir,
			ModelsDir: config.ModelsDir,
			ModelURL:  config.ModelURL,
		}, cache),
		router:        ui.NewRouter("/"),
		covers:        make(map[string]ui.ImageSource),
		progressCache: make(map[string]library.Progress),
		libraryChoice: -1,
	}
	application.reader = newReaderState()
	return application, nil
}

// Attach gives the app its window and saves progress when the window closes.
func (a *App) Attach(window *mygo.Window) {
	a.window = window
	window.OnClose(func(*mygo.CloseEvent) {
		a.Close()
	})
}

// Start begins loading the library. The loading screen shows until it is
// ready.
func (a *App) Start() {
	a.refresh(true)
}

// Close stops playback and saves the current book's progress.
func (a *App) Close() {
	a.closeSession(true)
}

// Update runs fn on the main thread and asks for a frame.
func (a *App) Update(fn func()) {
	if a.window != nil {
		a.window.Update(fn)
		return
	}
	fn()
}

// View builds the screen for the page the router shows.
func (a *App) View(c *ui.Context) {
	a.enforceRoute()
	a.router.View(c, func(route *ui.Route) {
		switch {
		case route.Match("/"):
			route.Title("Amba")
			a.loadingView(c)
		case route.Match("/library"):
			route.Title("Library")
			a.libraryView(c)
		case route.Match("/read/{id}"):
			route.Title(a.readerTitle())
			a.readerView(c, route.Param("id"))
		default:
			route.Title("Not found")
			a.notFoundView(c)
		}
	})
}

// enforceRoute stops the reader when the user leaves its page, as Back does.
func (a *App) enforceRoute() {
	if a.reader.status == readerIdle {
		return
	}
	if strings.HasPrefix(a.router.Path(), "/read/") {
		return
	}
	a.closeSession(true)
	a.reader = newReaderState()
}

func (a *App) refresh(initial bool) {
	if a.scanning {
		return
	}
	a.scanning = true
	a.scanStage = "Looking for audiobooks…"
	if initial {
		a.router.Replace("/")
	}

	go func() {
		books, err := library.Scan(a.config.LibraryRoot)
		a.Update(func() {
			a.books = books
			a.booksErr = err
			a.covers = decodeCovers(books)
			a.progressCache = a.loadProgress(books)
			a.scanning = false
			if initial {
				a.router.Replace("/library")
			}
		})
	}()
}

func (a *App) loadProgress(books []library.Book) map[string]library.Progress {
	progress := make(map[string]library.Progress, len(books))
	for _, book := range books {
		progress[book.ID] = a.progress.Load(book.ID)
	}
	return progress
}

func decodeCovers(books []library.Book) map[string]ui.ImageSource {
	covers := make(map[string]ui.ImageSource)
	for _, book := range books {
		if len(book.Cover) == 0 {
			continue
		}
		if image, err := ui.DecodeImage(book.Cover); err == nil {
			covers[book.ID] = image
		}
	}
	return covers
}

func (a *App) bookByID(id string) (library.Book, bool) {
	for _, book := range a.books {
		if book.ID == id {
			return book, true
		}
	}
	return library.Book{}, false
}

func (a *App) readerTitle() string {
	if a.reader.book.Title != "" {
		return a.reader.book.Title
	}
	return "Reading"
}

// projectRoot finds the folder holding the app's audiobooks and models, from
// the working directory or from beside the executable.
func projectRoot() string {
	if working, err := os.Getwd(); err == nil && hasProjectData(working) {
		return working
	}

	if executable, err := os.Executable(); err == nil {
		directory := filepath.Dir(executable)
		for range 6 {
			if hasProjectData(directory) {
				return directory
			}
			parent := filepath.Dir(directory)
			if parent == directory {
				break
			}
			directory = parent
		}
	}

	if working, err := os.Getwd(); err == nil {
		return working
	}
	return "."
}

func hasProjectData(directory string) bool {
	for _, name := range []string{"audiobooks", "models"} {
		if info, err := os.Stat(filepath.Join(directory, name)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}
