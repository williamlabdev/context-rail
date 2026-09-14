package registryhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"context-rail/internal/projectregistry"
)

// Server exposes the read-only Project Registry HTTP boundary.
type Server struct {
	roots   []string
	options ServerOptions
}

// ServerOptions controls deterministic local fixture scenarios for verification.
// It does not enable writes, remote discovery or arbitrary filesystem access.
type ServerOptions struct {
	Scenario string
	Delay    time.Duration
}

// NewServer creates a registry server with explicit, local Project roots.
func NewServer(roots []string) *Server {
	return NewServerWithOptions(roots, ServerOptions{})
}

// NewServerWithOptions creates a registry server with explicit local verification options.
func NewServerWithOptions(roots []string, options ServerOptions) *Server {
	return &Server{roots: append([]string(nil), roots...), options: options}
}

func (server *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This read-only registry accepts GET requests only.")
		return
	}

	const prefix = "/v1/projects"
	if request.URL.Path == prefix {
		server.writeList(response)
		return
	}
	if strings.HasPrefix(request.URL.Path, prefix+"/") {
		projectID, err := url.PathUnescape(strings.TrimPrefix(request.URL.Path, prefix+"/"))
		if err != nil || projectID == "" || strings.Contains(projectID, "/") {
			writeError(response, http.StatusNotFound, "PROJECT_NOT_FOUND", "The requested Project is not configured for this local registry.")
			return
		}
		server.writeDetail(response, projectID)
		return
	}
	writeError(response, http.StatusNotFound, "PROJECT_NOT_FOUND", "The requested Project is not configured for this local registry.")
}

func (server *Server) writeList(response http.ResponseWriter) {
	if server.options.Scenario == "empty" {
		writeJSON(response, http.StatusOK, projectregistry.Snapshot{
			Kind: "ProjectRegistrySnapshot", SchemaVersion: "project-registry/v1", Projects: []projectregistry.ProjectEntry{},
		})
		return
	}
	if server.writeScenarioError(response) {
		return
	}
	snapshot, err := server.importSnapshot()
	if err != nil {
		writeImportError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, snapshot)
}

func (server *Server) writeDetail(response http.ResponseWriter, projectID string) {
	if server.options.Scenario == "empty" {
		writeError(response, http.StatusNotFound, "PROJECT_NOT_FOUND", "The requested Project is not configured for this local registry.")
		return
	}
	if server.writeScenarioError(response) {
		return
	}
	snapshot, err := server.importSnapshot()
	if err != nil {
		writeImportError(response, err)
		return
	}
	for _, project := range snapshot.Projects {
		if project.Project.ID == projectID {
			snapshot.Projects = []projectregistry.ProjectEntry{project}
			writeJSON(response, http.StatusOK, snapshot)
			return
		}
	}
	writeError(response, http.StatusNotFound, "PROJECT_NOT_FOUND", "The requested Project is not configured for this local registry.")
}

func (server *Server) importSnapshot() (projectregistry.Snapshot, error) {
	if server.options.Delay > 0 {
		time.Sleep(server.options.Delay)
	}
	return projectregistry.Import(server.roots)
}

func (server *Server) writeScenarioError(response http.ResponseWriter) bool {
	switch server.options.Scenario {
	case "invalid":
		writeError(response, http.StatusUnprocessableEntity, "PROJECT_INVALID", "A configured Project is invalid and cannot be included in the read-only registry.")
		return true
	case "unavailable":
		writeError(response, http.StatusServiceUnavailable, "PROJECT_UNAVAILABLE", "A configured Project is unavailable to the local registry.")
		return true
	default:
		return false
	}
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeImportError(response http.ResponseWriter, err error) {
	message := "The local Project Registry could not be read."
	status := http.StatusServiceUnavailable
	code := "REGISTRY_UNAVAILABLE"
	if strings.Contains(err.Error(), "cannot parse project.yaml") || strings.Contains(err.Error(), "kind must be Project") || strings.Contains(err.Error(), "metadata.id and metadata.name") || strings.Contains(err.Error(), "document path") {
		status = http.StatusUnprocessableEntity
		code = "PROJECT_INVALID"
		message = "A configured Project is invalid and cannot be included in the read-only registry."
	} else if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "cannot read project.yaml") {
		code = "PROJECT_UNAVAILABLE"
		message = "A configured Project is unavailable to the local registry."
	}
	writeError(response, status, code, message)
}
