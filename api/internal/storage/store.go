// Package storage implements a minimal generic JSON-file-backed collection
// store with atomic (write-temp-then-rename) persistence. Each Store[T] owns
// one JSON file on disk and keeps a full in-memory copy, guarded by a mutex,
// since the whole application is a single process.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Store[T any] struct {
	mu   sync.Mutex
	path string
	data map[string]T
}

// NewStore loads an existing JSON file at path, or creates an empty one.
func NewStore[T any](path string) (*Store[T], error) {
	s := &Store[T]{path: path, data: make(map[string]T)}

	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if writeErr := atomicWriteJSON(path, s.data); writeErr != nil {
				return nil, writeErr
			}
			return s, nil
		}
		return nil, err
	}

	if len(bytes) == 0 {
		return s, nil
	}

	if err := json.Unmarshal(bytes, &s.data); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return s, nil
}

func (s *Store[T]) Get(id string) (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[id]
	return v, ok
}

func (s *Store[T]) List() []T {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]T, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, v)
	}
	return out
}

// Put upserts v under id and persists the whole collection atomically.
func (s *Store[T]) Put(id string, v T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, existed := s.data[id]
	s.data[id] = v
	if err := atomicWriteJSON(s.path, s.data); err != nil {
		if existed {
			s.data[id] = prev
		} else {
			delete(s.data, id)
		}
		return err
	}
	return nil
}

func (s *Store[T]) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, existed := s.data[id]
	if !existed {
		return nil
	}
	delete(s.data, id)
	if err := atomicWriteJSON(s.path, s.data); err != nil {
		s.data[id] = prev
		return err
	}
	return nil
}

// Find returns all entries matching pred, in no particular order. Always
// non-nil so JSON-encoding an empty result produces [] rather than null.
func (s *Store[T]) Find(pred func(T) bool) []T {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]T, 0)
	for _, v := range s.data {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

// atomicWriteJSON marshals v and writes it to path via a temp file in the
// same directory followed by an atomic rename, so a crash mid-write never
// leaves a partially-written file at path.
func atomicWriteJSON(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(bytes); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// WriteFileAtomic writes an arbitrary binary blob atomically (temp file in
// the same directory, then rename), used for attachment bytes.
func WriteFileAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
