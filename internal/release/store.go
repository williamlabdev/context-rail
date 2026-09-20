package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var projectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// Store persists one release ledger per Project.
type Store interface {
	Load(projectID string) (*State, error)
	Save(projectID string, state *State) error
}

// FileStore keeps releases.json per Project under dir with atomic writes.
type FileStore struct {
	dir string
	mu  sync.Mutex
}

func NewFileStore(dir string) (*FileStore, error) {
	if dir == "" {
		return nil, errors.New("state directory is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

func (store *FileStore) path(projectID string) (string, error) {
	if !projectIDPattern.MatchString(projectID) {
		return "", newError(CodeProjectNotFound, "invalid project id %q", projectID)
	}
	return filepath.Join(store.dir, projectID, "releases.json"), nil
}

func (store *FileStore) Load(projectID string) (*State, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	path, err := store.path(projectID)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, newError(CodeStateUnavailable, "read release ledger: %v", err)
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, newError(CodeStateUnavailable, "decode release ledger: %v", err)
	}
	if state.Kind != Kind || state.SchemaVersion != SchemaVersion {
		return nil, newError(CodeStateUnavailable, "unexpected ledger kind %q/%q", state.Kind, state.SchemaVersion)
	}
	return &state, nil
}

func (store *FileStore) Save(projectID string, state *State) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	path, err := store.path(projectID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return newError(CodeStateUnavailable, "create project state directory: %v", err)
	}
	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return newError(CodeStateUnavailable, "encode release ledger: %v", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "releases-*.json.tmp")
	if err != nil {
		return newError(CodeStateUnavailable, "create temp ledger: %v", err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(encoded, '\n')); err != nil {
		tmp.Close()
		os.Remove(name)
		return newError(CodeStateUnavailable, "write temp ledger: %v", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(name)
		return newError(CodeStateUnavailable, "sync temp ledger: %v", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return newError(CodeStateUnavailable, "close temp ledger: %v", err)
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return newError(CodeStateUnavailable, "commit ledger: %v", err)
	}
	return nil
}

// MemoryStore is for tests.
type MemoryStore struct {
	mu     sync.Mutex
	states map[string][]byte
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{states: map[string][]byte{}} }

func (store *MemoryStore) Load(projectID string) (*State, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	raw, ok := store.states[projectID]
	if !ok {
		return nil, nil
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, newError(CodeStateUnavailable, "decode release ledger: %v", err)
	}
	return &state, nil
}

func (store *MemoryStore) Save(projectID string, state *State) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	raw, err := json.Marshal(state)
	if err != nil {
		return newError(CodeStateUnavailable, "encode release ledger: %v", err)
	}
	store.states[projectID] = raw
	return nil
}
