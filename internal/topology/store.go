package topology

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

// Store persists one State per Project. Implementations must be atomic per
// Save so a crash never leaves a half-written topology.
type Store interface {
	Load(projectID string) (*State, error) // returns nil, nil when no state exists
	Save(projectID string, state *State) error
}

var projectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// FileStore keeps one JSON document per Project under dir. It is the P0
// persistence decision (files first, Firestore later); the document is the
// full versioned history, never a partial update.
type FileStore struct {
	dir string
	mu  sync.Mutex
}

// NewFileStore creates the directory if needed.
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
	return filepath.Join(store.dir, projectID, "topology.json"), nil
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
		return nil, newError(CodeStateUnavailable, "read topology state: %v", err)
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, newError(CodeStateUnavailable, "decode topology state: %v", err)
	}
	if state.Kind != Kind || state.SchemaVersion != SchemaVersion {
		return nil, newError(CodeStateUnavailable, "unexpected topology state kind %q/%q", state.Kind, state.SchemaVersion)
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
		return newError(CodeStateUnavailable, "encode topology state: %v", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "topology-*.json.tmp")
	if err != nil {
		return newError(CodeStateUnavailable, "create temp state file: %v", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(append(encoded, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return newError(CodeStateUnavailable, "write temp state file: %v", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return newError(CodeStateUnavailable, "sync temp state file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return newError(CodeStateUnavailable, "close temp state file: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return newError(CodeStateUnavailable, "commit state file: %v", err)
	}
	return nil
}

// MemoryStore is an in-process store for tests and throwaway runs.
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
		return nil, newError(CodeStateUnavailable, "decode topology state: %v", err)
	}
	return &state, nil
}

func (store *MemoryStore) Save(projectID string, state *State) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	raw, err := json.Marshal(state)
	if err != nil {
		return newError(CodeStateUnavailable, "encode topology state: %v", err)
	}
	store.states[projectID] = raw
	return nil
}
