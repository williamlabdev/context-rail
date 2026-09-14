package registryhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	"context-rail/internal/projectregistry"
)

// Server exposes the read-only Project Registry HTTP boundary.
type Server struct {
	roots []string
}

// NewServer creates a registry server with explicit, local Project roots.
func NewServer(roots []string) *Server {
	return &Server{roots: append([]string(nil), roots...)}
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
	snapshot, err := projectregistry.Import(server.roots)
	if err != nil {
		writeImportError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, snapshot)
}

func (server *Server) writeDetail(response http.ResponseWriter, projectID string) {
	snapshot, err := projectregistry.Import(server.roots)
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
