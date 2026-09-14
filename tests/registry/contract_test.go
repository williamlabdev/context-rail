package registry_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"context-rail/internal/projectregistry"
)

func workspaceRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func fixtureRoots(t *testing.T) []string {
	root := workspaceRoot(t)
	return []string{
		filepath.Join(root, "demo", "order-operations-portal"),
		filepath.Join(root, "examples", "support-insights"),
	}
}

func projectByID(t *testing.T, snapshot projectregistry.Snapshot, id string) projectregistry.ProjectEntry {
	t.Helper()
	for _, project := range snapshot.Projects {
		if project.Project.ID == id {
			return project
		}
	}
	t.Fatalf("project %q not found", id)
	return projectregistry.ProjectEntry{}
}

func documentByPath(t *testing.T, project projectregistry.ProjectEntry, path string) projectregistry.DocumentStatus {
	t.Helper()
	for _, document := range project.Documents {
		if document.Path == path {
			return document
		}
	}
	t.Fatalf("document %q not found", path)
	return projectregistry.DocumentStatus{}
}

func TestImportSnapshotIncludesBothFixtureShapes(t *testing.T) {
	snapshot, err := projectregistry.Import(fixtureRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Kind != "ProjectRegistrySnapshot" {
		t.Fatalf("kind = %q", snapshot.Kind)
	}
	if snapshot.SchemaVersion != "project-registry/v1" {
		t.Fatalf("schema version = %q", snapshot.SchemaVersion)
	}
	if len(snapshot.Projects) != 2 {
		t.Fatalf("project count = %d", len(snapshot.Projects))
	}

	order := projectByID(t, snapshot, "order-operations-portal")
	if order.Project.Name != "Order Operations Portal" || order.Project.Purpose == "" {
		t.Fatalf("incomplete order project identity: %#v", order.Project)
	}
	if len(order.Repositories) != 1 || order.Repositories[0].Status != "fixture-pending-remote" {
		t.Fatalf("unexpected order repositories: %#v", order.Repositories)
	}
	if len(order.Environments) != 4 || order.Environments[0].ID != "development" || order.Environments[3].ID != "production" {
		t.Fatalf("unexpected order environments: %#v", order.Environments)
	}
	if order.Context.Status != "DERIVED" || order.Readiness.Production.Status != "BLOCKED" {
		t.Fatalf("unexpected order context/readiness: %#v %#v", order.Context, order.Readiness)
	}

	support := projectByID(t, snapshot, "support-insights")
	if len(support.Services) != 1 || support.Services[0].Runtime != "UNDECLARED" {
		t.Fatalf("unexpected support service normalization: %#v", support.Services)
	}
	if documentByPath(t, support, "docs/ai/context-pack.json").Status != "STALE" {
		t.Fatalf("support Context Pack is not stale")
	}
	if len(documentByPath(t, support, "docs/ai/context-pack.json").StaleSources) == 0 {
		t.Fatalf("support stale document has no stale source provenance")
	}
	if !snapshot.Projects[0].ReadOnly || !snapshot.Projects[1].ReadOnly {
		t.Fatal("snapshot must be explicitly read-only")
	}
}

func TestImportNormalizesSingularAndPluralServices(t *testing.T) {
	snapshot, err := projectregistry.Import(fixtureRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	order := projectByID(t, snapshot, "order-operations-portal")
	support := projectByID(t, snapshot, "support-insights")
	if len(order.Services) != 1 || order.Services[0].ID != "order-operations-web" || order.Services[0].Path != "." {
		t.Fatalf("singular service was not normalized: %#v", order.Services)
	}
	if len(support.Services) != 1 || support.Services[0].Repository == nil || *support.Services[0].Repository != "support-insights" {
		t.Fatalf("plural service was not normalized: %#v", support.Services)
	}
}

func TestImportPreservesUncertainStatuses(t *testing.T) {
	snapshot, err := projectregistry.Import(fixtureRoots(t))
	if err != nil {
		t.Fatal(err)
	}
	support := projectByID(t, snapshot, "support-insights")
	context := documentByPath(t, support, "docs/ai/context-pack.json")
	if context.Status == "CURRENT" || context.Status == "PASS" {
		t.Fatalf("stale context was upgraded: %#v", context)
	}
	if support.Services[0].Runtime == "" || support.Services[0].Runtime == "PASS" {
		t.Fatalf("undeclared runtime was erased or upgraded: %#v", support.Services[0])
	}
	if support.Readiness.Production.Status != "BLOCKED" {
		t.Fatalf("production gate was not preserved: %#v", support.Readiness.Production)
	}
}

func TestImportRejectsInvalidManifestWithoutPartialSnapshot(t *testing.T) {
	invalidRoot := filepath.Join(workspaceRoot(t), "tests", "registry", "fixtures", "invalid")
	snapshot, err := projectregistry.Import([]string{invalidRoot})
	if err == nil {
		t.Fatal("invalid manifest unexpectedly imported")
	}
	if snapshot.Kind != "" || len(snapshot.Projects) != 0 {
		t.Fatalf("partial snapshot returned with error: %#v", snapshot)
	}
	if !strings.Contains(err.Error(), invalidRoot) {
		t.Fatalf("error does not identify root %q: %v", invalidRoot, err)
	}
}

func TestImportRejectsMissingManifest(t *testing.T) {
	missingRoot := filepath.Join(workspaceRoot(t), "tests", "registry", "fixtures", "missing")
	_, err := projectregistry.Import([]string{missingRoot})
	if err == nil {
		t.Fatal("missing manifest unexpectedly imported")
	}
	if !strings.Contains(err.Error(), missingRoot) || !strings.Contains(err.Error(), "project.yaml") {
		t.Fatalf("error does not identify missing manifest: %v", err)
	}
}
