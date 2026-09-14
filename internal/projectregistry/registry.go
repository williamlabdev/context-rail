package projectregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Import reads the explicitly supplied Project roots and returns one atomic snapshot.
// It never writes to a consumer Project or to a governance artifact.
func Import(roots []string) (Snapshot, error) {
	if len(roots) == 0 {
		return Snapshot{}, fmt.Errorf("at least one Project root is required")
	}
	snapshot := Snapshot{Kind: "ProjectRegistrySnapshot", SchemaVersion: "project-registry/v1", Projects: make([]ProjectEntry, 0, len(roots))}
	for _, inputRoot := range roots {
		root, err := filepath.Abs(inputRoot)
		if err != nil {
			return Snapshot{}, fmt.Errorf("%s: cannot resolve Project root: %w", inputRoot, err)
		}
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			if os.IsNotExist(err) {
				return Snapshot{}, fmt.Errorf("%s: cannot read project.yaml: %w", inputRoot, err)
			}
			return Snapshot{}, fmt.Errorf("%s: cannot resolve Project root: %w", inputRoot, err)
		}
		manifest, err := readManifest(root)
		if err != nil {
			return Snapshot{}, err
		}
		documents, err := documentStatuses(root, manifest.Spec.Documents)
		if err != nil {
			return Snapshot{}, err
		}
		decisions, rawDecisions, err := readDecisions(root)
		if err != nil {
			return Snapshot{}, err
		}
		readiness, err := readinessFor(root, manifest.Spec, documents, decisions, rawDecisions)
		if err != nil {
			return Snapshot{}, err
		}
		status := manifest.Metadata.Status
		if status == "" {
			status = "UNDECLARED"
		}
		owners := manifest.Spec.Owners
		if owners == nil {
			owners = map[string]string{}
		}
		snapshot.Projects = append(snapshot.Projects, ProjectEntry{
			Project: ProjectRecord{
				ID: manifest.Metadata.ID, Name: manifest.Metadata.Name, Version: manifest.Metadata.Version,
				Classification: manifest.Metadata.Classification, Status: status, Purpose: manifest.Spec.Purpose,
				Owners: owners, Root: root,
			},
			Repositories: normalizeRepositories(manifest.Spec),
			Services:     normalizeServices(manifest.Spec),
			Environments: normalizeEnvironments(manifest.Spec.Environments),
			Documents:    documents,
			Decisions:    decisions,
			Readiness:    readiness,
			Context:      contextStatus(documents),
			ReadOnly:     true,
			ObservedAt:   time.Now().UTC().Truncate(time.Second).Format(time.RFC3339),
		})
	}
	return snapshot, nil
}

func contextStatus(documents []DocumentStatus) ContextStatus {
	for _, document := range documents {
		if document.Path == "docs/ai/context-pack.json" {
			return ContextStatus{Status: document.Status}
		}
	}
	return ContextStatus{Status: "MISSING"}
}

func readDecisions(root string) ([]DecisionSummary, []map[string]any, error) {
	directory := filepath.Join(root, "decisions")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []DecisionSummary{}, []map[string]any{}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%s: cannot read decisions: %w", root, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	summaries := []DecisionSummary{}
	raw := []map[string]any{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var value map[string]any
		if json.Unmarshal(data, &value) != nil {
			continue
		}
		raw = append(raw, value)
		summaries = append(summaries, DecisionSummary{
			DecisionID: stringValue(value["decision_id"]),
			RequestID:  stringValue(value["request_id"]),
			Status:     stringValue(value["status"]),
		})
	}
	return summaries, raw, nil
}

func stringValue(value any) string {
	valueString, _ := value.(string)
	return valueString
}
