package registryhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"context-rail/internal/change"
)

// ChangesHandler exposes the Change Decision Pack path of a Project.
//
//	GET  /v1/projects/{project}/changes
//	POST /v1/projects/{project}/changes                      open a Change (evaluated immediately)
//	GET  /v1/projects/{project}/changes/{change}             change + decision + brief + pack + work order
//	POST /v1/projects/{project}/changes/{change}/inputs      supply inputs → new version, re-evaluated
//	POST /v1/projects/{project}/changes/{change}/decision    human ACCEPT / REJECT
//	POST /v1/projects/{project}/changes/{change}/work-order  compile hashed Agent Work Order
type ChangesHandler struct {
	service *change.Service
}

func NewChangesHandler(service *change.Service) *ChangesHandler {
	return &ChangesHandler{service: service}
}

func (handler *ChangesHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/projects/{project}/changes", handler.list)
	mux.HandleFunc("POST /v1/projects/{project}/changes", handler.create)
	mux.HandleFunc("GET /v1/projects/{project}/changes/{change}", handler.get)
	mux.HandleFunc("POST /v1/projects/{project}/changes/{change}/inputs", handler.inputs)
	mux.HandleFunc("POST /v1/projects/{project}/changes/{change}/decision", handler.decide)
	mux.HandleFunc("POST /v1/projects/{project}/changes/{change}/work-order", handler.workOrder)
	mux.HandleFunc("/v1/projects/{project}/changes", methodNotAllowed("GET, POST"))
	mux.HandleFunc("/v1/projects/{project}/changes/{change}", methodNotAllowed("GET"))
}

type changeList struct {
	Kind          string        `json:"kind"`
	SchemaVersion string        `json:"schema_version"`
	ProjectID     string        `json:"project_id"`
	Changes       []change.View `json:"changes"`
}

func (handler *ChangesHandler) list(response http.ResponseWriter, request *http.Request) {
	projectID := request.PathValue("project")
	views, err := handler.service.List(projectID)
	if err != nil {
		writeChangeError(response, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, changeList{Kind: "ChangeList", SchemaVersion: change.SchemaVersion, ProjectID: projectID, Changes: views})
}

func (handler *ChangesHandler) get(response http.ResponseWriter, request *http.Request) {
	view, err := handler.service.Get(request.PathValue("project"), request.PathValue("change"))
	handler.respond(response, view, err)
}

func (handler *ChangesHandler) create(response http.ResponseWriter, request *http.Request) {
	var body change.CreateRequest
	if !decodeChangeBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.Create(request.PathValue("project"), body)
	handler.respond(response, view, err)
}

func (handler *ChangesHandler) inputs(response http.ResponseWriter, request *http.Request) {
	var body change.InputsRequest
	if !decodeChangeBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.SupplyInputs(request.PathValue("project"), request.PathValue("change"), body)
	handler.respond(response, view, err)
}

func (handler *ChangesHandler) decide(response http.ResponseWriter, request *http.Request) {
	var body change.DecideRequest
	if !decodeChangeBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.Decide(request.PathValue("project"), request.PathValue("change"), body)
	handler.respond(response, view, err)
}

func (handler *ChangesHandler) workOrder(response http.ResponseWriter, request *http.Request) {
	var body change.WorkOrderRequest
	if !decodeChangeBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.CompileWorkOrder(request.PathValue("project"), request.PathValue("change"), body)
	handler.respond(response, view, err)
}

func (handler *ChangesHandler) respond(response http.ResponseWriter, view *change.View, err error) {
	if err != nil {
		writeChangeError(response, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, view)
}

func decodeChangeBody(response http.ResponseWriter, request *http.Request, body any, mutation *change.Mutation) bool {
	if ct := request.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeError(response, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Change mutations accept application/json bodies only.")
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

func writeChangeError(response http.ResponseWriter, err error) {
	var typed *change.Error
	if !errors.As(err, &typed) {
		// Topology errors can surface through the facts source; keep their code.
		writeTopologyError(response, err)
		return
	}
	status := http.StatusUnprocessableEntity
	switch typed.Code {
	case change.CodeProjectNotFound, change.CodeChangeNotFound:
		status = http.StatusNotFound
	case change.CodeAlreadyDecided, change.CodeChangeClosed, change.CodeDecisionStale, change.CodeAlreadyIssued, change.CodeNotAccepted:
		status = http.StatusConflict
	case change.CodeStateUnavailable:
		status = http.StatusServiceUnavailable
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(struct {
		Kind    string `json:"kind"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Kind: "ChangeError", Code: typed.Code, Message: typed.Message})
}
