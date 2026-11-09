package store

import (
	"os"
	"path/filepath"
	"testing"
)

type record struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestWriteAndReadJSON(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	filename := store.File("nested", "record.json")
	if err := store.WriteJSON(filename, record{Name: "book", Count: 3}); err != nil {
		t.Fatal(err)
	}

	var got record
	if err := store.ReadJSON(filename, &got); err != nil {
		t.Fatal(err)
	}
	if got != (record{Name: "book", Count: 3}) {
		t.Fatalf("ReadJSON() = %+v, want the record written", got)
	}

	// Writing again replaces the file rather than appending to it.
	if err := store.WriteJSON(filename, record{Name: "next", Count: 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReadJSON(filename, &got); err != nil {
		t.Fatal(err)
	}
	if got != (record{Name: "next", Count: 1}) {
		t.Fatalf("ReadJSON() = %+v, want the replacement", got)
	}
}

func TestReadJSONMissingIsAnError(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	var got record
	if err := store.ReadJSON(store.File("missing.json"), &got); err == nil {
		t.Fatal("ReadJSON() of a missing file returned no error")
	}
}

func TestNewCreatesRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache", "Amba")
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if store.Root != root {
		t.Fatalf("Root = %q, want %q", store.Root, root)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		t.Fatalf("New() did not create the cache directory: %v", err)
	}
}
