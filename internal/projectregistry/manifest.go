package projectregistry

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type manifest struct {
	Kind     string           `yaml:"kind"`
	Metadata manifestMetadata `yaml:"metadata"`
	Spec     manifestSpec     `yaml:"spec"`
}

type manifestMetadata struct {
	ID             string `yaml:"id"`
	Name           string `yaml:"name"`
	Version        int    `yaml:"version"`
	Classification string `yaml:"classification"`
	Status         string `yaml:"status"`
}

type manifestSpec struct {
	Purpose      string                `yaml:"purpose"`
	Owners       map[string]string     `yaml:"owners"`
	Repository   *manifestRepository   `yaml:"repository"`
	Repositories []manifestRepository  `yaml:"repositories"`
	Service      *manifestService      `yaml:"service"`
	Services     []manifestService     `yaml:"services"`
	Environments []manifestEnvironment `yaml:"environments"`
	Documents    []manifestDocument    `yaml:"documents"`
}

type manifestRepository struct {
	Provider      string `yaml:"provider"`
	Visibility    string `yaml:"visibility"`
	URL           string `yaml:"url"`
	DefaultBranch string `yaml:"defaultBranch"`
	Status        string `yaml:"status"`
}

type manifestService struct {
	ID         string  `yaml:"id"`
	Repository *string `yaml:"repository"`
	Path       string  `yaml:"path"`
	SourcePath string  `yaml:"sourcePath"`
	Runtime    string  `yaml:"runtime"`
	Role       string  `yaml:"role"`
}

type manifestEnvironment struct {
	ID               string   `yaml:"id"`
	Type             string   `yaml:"type"`
	Sequence         int      `yaml:"sequence"`
	TargetRef        string   `yaml:"targetRef"`
	RequiredEvidence []string `yaml:"requiredEvidence"`
	Action           *string  `yaml:"action"`
}

type manifestDocument struct {
	Path          string   `yaml:"path"`
	Kind          string   `yaml:"kind"`
	SourceOfTruth *bool    `yaml:"sourceOfTruth"`
	Status        string   `yaml:"status"`
	RequiredFor   []string `yaml:"requiredFor"`
}

func readManifest(root string) (manifest, error) {
	path := filepath.Join(root, "project.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("%s: cannot read project.yaml: %w", root, err)
	}
	var value manifest
	if err := yaml.Unmarshal(data, &value); err != nil {
		return manifest{}, fmt.Errorf("%s: cannot parse project.yaml: %w", root, err)
	}
	if value.Kind != "Project" {
		return manifest{}, fmt.Errorf("%s: project.yaml kind must be Project", root)
	}
	if value.Metadata.ID == "" || value.Metadata.Name == "" {
		return manifest{}, fmt.Errorf("%s: project.yaml metadata.id and metadata.name are required", root)
	}
	return value, nil
}

func normalizeRepositories(spec manifestSpec) []Repository {
	values := make([]manifestRepository, 0, len(spec.Repositories)+1)
	if spec.Repository != nil {
		values = append(values, *spec.Repository)
	}
	values = append(values, spec.Repositories...)
	repositories := make([]Repository, 0, len(values))
	for _, value := range values {
		status := value.Status
		if status == "" {
			status = "UNDECLARED"
		}
		repositories = append(repositories, Repository{
			Provider: value.Provider, Visibility: value.Visibility, URL: value.URL,
			DefaultBranch: value.DefaultBranch, Status: status,
		})
	}
	return repositories
}

func normalizeServices(spec manifestSpec) []Service {
	values := make([]manifestService, 0, len(spec.Services)+1)
	if spec.Service != nil {
		values = append(values, *spec.Service)
	}
	values = append(values, spec.Services...)
	services := make([]Service, 0, len(values))
	for _, value := range values {
		path := value.Path
		if path == "" {
			path = value.SourcePath
		}
		runtime := value.Runtime
		if runtime == "" {
			runtime = "UNDECLARED"
		}
		role := value.Role
		if role == "" {
			role = "primary"
		}
		services = append(services, Service{ID: value.ID, Repository: value.Repository, Path: path, Runtime: runtime, Role: role})
	}
	return services
}

func normalizeEnvironments(values []manifestEnvironment) []Environment {
	environments := make([]Environment, 0, len(values))
	for _, value := range values {
		required := append([]string(nil), value.RequiredEvidence...)
		environments = append(environments, Environment{
			ID: value.ID, Type: value.Type, Sequence: value.Sequence, TargetRef: value.TargetRef,
			RequiredEvidence: required, Action: value.Action,
		})
	}
	return environments
}

func sourceOfTruth(value manifestDocument) bool {
	return value.SourceOfTruth == nil || *value.SourceOfTruth
}

func safeDocumentPath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%s: document path must be relative", root)
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: document path escapes Project root: %s", root, relative)
	}
	return filepath.Join(root, clean), nil
}

func documentStatuses(root string, values []manifestDocument) ([]DocumentStatus, error) {
	documents := make([]DocumentStatus, 0, len(values))
	for _, value := range values {
		path, err := safeDocumentPath(root, value.Path)
		if err != nil {
			return nil, err
		}
		status := value.Status
		if status == "" {
			status = "MISSING"
			if info, statErr := os.Stat(path); statErr == nil && info.Mode().IsRegular() {
				status = "CURRENT"
			}
		}
		document := DocumentStatus{Path: value.Path, Kind: value.Kind, SourceOfTruth: sourceOfTruth(value), Status: status}
		if !document.SourceOfTruth && status == "CURRENT" {
			derived, staleSources, err := derivedStatus(path, root)
			if err != nil {
				return nil, err
			}
			document.Status = derived
			document.StaleSources = staleSources
		}
		documents = append(documents, document)
	}
	return documents, nil
}

func derivedStatus(path, root string) (string, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "NEEDS_INPUT", nil, fmt.Errorf("%s: cannot read derived document: %w", path, err)
	}
	var pack struct {
		Derived bool `json:"derived"`
		Sources []struct {
			Path        string `json:"path"`
			ContentHash string `json:"content_hash"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(data, &pack); err != nil {
		return "NEEDS_INPUT", nil, nil
	}
	status := "NEEDS_INPUT"
	if pack.Derived {
		status = "DERIVED"
	}
	var stale []string
	for _, source := range pack.Sources {
		sourcePath, err := safeDocumentPath(root, source.Path)
		if err != nil {
			return "NEEDS_INPUT", nil, err
		}
		if info, statErr := os.Stat(sourcePath); statErr == nil && info.Mode().IsRegular() {
			actual, hashErr := fileHash(sourcePath)
			if hashErr != nil {
				return "NEEDS_INPUT", nil, hashErr
			}
			if source.ContentHash != "sha256:"+actual {
				stale = append(stale, source.Path)
			}
		}
	}
	if len(stale) > 0 {
		status = "STALE"
	}
	return status, stale, nil
}

func fileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// DeclaredDocument is one document as the Project manifest declares it,
// including the readiness stages it is required for. It is read-only input
// for the governed document baseline (UI-20); the registry never edits it.
type DeclaredDocument struct {
	Path          string   `json:"path"`
	Kind          string   `json:"kind"`
	SourceOfTruth bool     `json:"source_of_truth"`
	RequiredFor   []string `json:"required_for"`
	DeclaredState string   `json:"declared_state,omitempty"`
}

// DeclaredDocuments reads the document declarations of the manifest at root.
func DeclaredDocuments(root string) ([]DeclaredDocument, error) {
	value, err := readManifest(root)
	if err != nil {
		return nil, err
	}
	documents := make([]DeclaredDocument, 0, len(value.Spec.Documents))
	for _, document := range value.Spec.Documents {
		documents = append(documents, DeclaredDocument{
			Path: document.Path, Kind: document.Kind, SourceOfTruth: sourceOfTruth(document),
			RequiredFor: append([]string{}, document.RequiredFor...), DeclaredState: document.Status,
		})
	}
	return documents, nil
}
