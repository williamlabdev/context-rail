package registryhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"context-rail/internal/topology"
)

// ActorHeader carries the operator identity for topology mutations in P0.
// There is no authentication yet; the value is recorded in the audit trail as
// declared, never treated as authorization.
const ActorHeader = "X-ContextRail-Actor"

const maxMutationBody = 64 * 1024

// TopologyHandler exposes the versioned Environment Topology of a Project.
//
//	GET   /v1/projects/{project}/environments
//	POST  /v1/projects/{project}/environments              add
//	POST  /v1/projects/{project}/environments/reorder      reorder
//	PATCH /v1/projects/{project}/environments/{env}        edit
//	POST  /v1/projects/{project}/environments/{env}/retire
//	POST  /v1/projects/{project}/environments/{env}/restore
type TopologyHandler struct {
	service *topology.Service
}

func NewTopologyHandler(service *topology.Service) *TopologyHandler {
	return &TopologyHandler{service: service}
}

// Register mounts the routes on a Go 1.22+ pattern mux.
func (handler *TopologyHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/projects/{project}/environments", handler.get)
	mux.HandleFunc("POST /v1/projects/{project}/environments", handler.add)
	mux.HandleFunc("POST /v1/projects/{project}/environments/reorder", handler.reorder)
	mux.HandleFunc("PATCH /v1/projects/{project}/environments/{env}", handler.edit)
	mux.HandleFunc("POST /v1/projects/{project}/environments/{env}/retire", handler.retire)
	mux.HandleFunc("POST /v1/projects/{project}/environments/{env}/restore", handler.restore)
	// Any other method on these paths is a deliberate 405, not a fallthrough
	// into the read-only registry handler.
	mux.HandleFunc("/v1/projects/{project}/environments", methodNotAllowed("GET, POST"))
	mux.HandleFunc("/v1/projects/{project}/environments/{env}", methodNotAllowed("PATCH"))
}

func (handler *TopologyHandler) get(response http.ResponseWriter, request *http.Request) {
	state, err := handler.service.Get(request.PathValue("project"))
	handler.respond(response, state, err)
}

func (handler *TopologyHandler) add(response http.ResponseWriter, request *http.Request) {
	var body topology.AddRequest
	if !decodeMutation(response, request, &body, &body.Mutation) {
		return
	}
	state, err := handler.service.Add(request.PathValue("project"), body)
	handler.respond(response, state, err)
}

func (handler *TopologyHandler) edit(response http.ResponseWriter, request *http.Request) {
	var body topology.EditRequest
	if !decodeMutation(response, request, &body, &body.Mutation) {
		return
	}
	state, err := handler.service.Edit(request.PathValue("project"), request.PathValue("env"), body)
	handler.respond(response, state, err)
}

func (handler *TopologyHandler) reorder(response http.ResponseWriter, request *http.Request) {
	var body topology.ReorderRequest
	if !decodeMutation(response, request, &body, &body.Mutation) {
		return
	}
	state, err := handler.service.Reorder(request.PathValue("project"), body)
	handler.respond(response, state, err)
}

func (handler *TopologyHandler) retire(response http.ResponseWriter, request *http.Request) {
	var body topology.Mutation
	if !decodeMutation(response, request, &body, &body) {
		return
	}
	state, err := handler.service.Retire(request.PathValue("project"), request.PathValue("env"), body)
	handler.respond(response, state, err)
}

func (handler *TopologyHandler) restore(response http.ResponseWriter, request *http.Request) {
	var body topology.Mutation
	if !decodeMutation(response, request, &body, &body) {
		return
	}
	state, err := handler.service.Restore(request.PathValue("project"), request.PathValue("env"), body)
	handler.respond(response, state, err)
}

func (handler *TopologyHandler) respond(response http.ResponseWriter, state *topology.State, err error) {
	if err != nil {
		writeTopologyError(response, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, state)
}

// decodeMutation parses a JSON body and fills the actor from the header when
// the body does not declare one.
func decodeMutation(response http.ResponseWriter, request *http.Request, body any, mutation *topology.Mutation) bool {
	if ct := request.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeError(response, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Topology mutations accept application/json bodies only.")
		return false
	}
	decoder := json.NewDecoder(io.LimitReader(request.Body, maxMutationBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(body); err != nil {
		writeError(response, http.StatusBadRequest, "INVALID_MUTATION_BODY", "The mutation body could not be parsed: "+err.Error())
		return false
	}
	if mutation.Actor == "" {
		mutation.Actor = strings.TrimSpace(request.Header.Get(ActorHeader))
	}
	return true
}

func writeTopologyError(response http.ResponseWriter, err error) {
	var typed *topology.Error
	if !errors.As(err, &typed) {
		writeError(response, http.StatusInternalServerError, "TOPOLOGY_ERROR", err.Error())
		return
	}
	status := http.StatusUnprocessableEntity
	switch typed.Code {
	case topology.CodeProjectNotFound, topology.CodeEnvironmentNotFound:
		status = http.StatusNotFound
	case topology.CodeVersionConflict, topology.CodeEnvironmentExists, topology.CodeProductionProtected,
		topology.CodeEnvironmentRetired, topology.CodeEnvironmentActive:
		status = http.StatusConflict
	case topology.CodeStateUnavailable:
		status = http.StatusServiceUnavailable
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(struct {
		Kind    string `json:"kind"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Kind: "EnvironmentTopologyError", Code: typed.Code, Message: typed.Message})
}

func methodNotAllowed(allow string) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Allow", allow)
		writeError(response, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "This topology route accepts "+allow+" only.")
	}
}
