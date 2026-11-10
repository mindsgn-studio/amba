// Package store keeps the app's cache files in one place and writes JSON
// atomically, so a crash never leaves half a file behind.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Store is a directory of cache files.
type Store struct {
	Root string
}

// New creates a store rooted at root.
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}
	return &Store{Root: root}, nil
}

// File returns a path inside the store.
func (s *Store) File(parts ...string) string {
	all := append([]string{s.Root}, parts...)
	return filepath.Join(all...)
}

// ReadJSON reads a JSON file into value. It returns an error when the file is
// missing or invalid, which callers treat as a cache miss.
func (s *Store) ReadJSON(filename string, value any) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

// WriteJSON writes value as indented JSON, replacing filename atomically.
func (s *Store) WriteJSON(filename string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(filename), ".tmp-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, filename)
}

// Remove deletes a file from the store. A missing file is not an error, so
// callers can remove an optional record without checking for it first.
func (s *Store) Remove(filename string) error {
	err := os.Remove(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
