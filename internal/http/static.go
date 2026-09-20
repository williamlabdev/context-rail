package registryhttp

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// healthResponse is the minimal liveness payload for the Cloud Run baseline.
// It reports process liveness only; it is not a readiness, authorization or
// deployment claim for any governed Project.
type healthResponse struct {
	Kind     string            `json:"kind"`
	Status   string            `json:"status"`
	Mode     string            `json:"mode"`
	Surfaces map[string]string `json:"surfaces"`
}

// NewHealthHandler returns a GET-only liveness endpoint.
func NewHealthHandler(mode string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.Header().Set("Allow", http.MethodGet)
			writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The health endpoint accepts GET requests only.")
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("Cache-Control", "no-store")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(healthResponse{
			Kind: "ContextRailHealth", Status: "ok", Mode: mode,
			Surfaces: map[string]string{"registry": "read-only", "topology": "versioned-writes", "changes": "governed-ledger", "releases": "promotion-gate"},
		})
	})
}

// NewStaticHandler serves the built workspace from dir with single-page
// fallback: a request for an unknown path that looks like a route (no file
// extension) receives index.html so the React workspace can render its own
// state. Requests for missing assets (paths with an extension) return 404.
// Only GET and HEAD are accepted; the workspace remains read-only.
func NewStaticHandler(dir string) http.Handler {
	root := filepath.Clean(dir)
	fileServer := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			response.Header().Set("Allow", "GET, HEAD")
			writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The read-only workspace accepts GET requests only.")
			return
		}
		cleaned := path.Clean("/" + request.URL.Path)
		if cleaned == "/" || cleaned == "/index.html" {
			serveIndex(response, request, root)
			return
		}
		candidate := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(cleaned, "/")))
		if !strings.HasPrefix(candidate, root+string(filepath.Separator)) {
			writeError(response, http.StatusNotFound, "ASSET_NOT_FOUND", "The requested asset is outside the workspace.")
			return
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(response, request)
			return
		}
		if path.Ext(cleaned) == "" {
			serveIndex(response, request, root)
			return
		}
		writeError(response, http.StatusNotFound, "ASSET_NOT_FOUND", "The requested asset does not exist in the built workspace.")
	})
}

func serveIndex(response http.ResponseWriter, request *http.Request, root string) {
	index := filepath.Join(root, "index.html")
	file, err := os.Open(index)
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, "WORKSPACE_NOT_BUILT", "The workspace static bundle is not present; run the frontend build before serving.")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		writeError(response, http.StatusServiceUnavailable, "WORKSPACE_NOT_BUILT", "The workspace static bundle is not readable.")
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	// ServeContent (not ServeFile) avoids the implicit /index.html → ./ redirect.
	http.ServeContent(response, request, "index.html", info.ModTime(), file)
}
