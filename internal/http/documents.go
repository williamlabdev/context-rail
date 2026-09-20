package registryhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"context-rail/internal/document"
)

// DocumentsHandler exposes the governed document baseline and the Context
// Pack rebuild (UI-20).
//
//	GET  /v1/projects/{project}/documents                    baseline, live statuses, readiness by stage, latest pack
//	POST /v1/projects/{project}/documents                    declare a required document (new baseline version)
//	POST /v1/projects/{project}/documents/withdraw           withdraw an operator declaration (new baseline version)
//	POST /v1/projects/{project}/context-pack/rebuild         rebuild the Context Pack from the sources as they are now
//	GET  /v1/projects/{project}/context-pack/{pack}          one rebuilt pack
type DocumentsHandler struct {
	service *document.Service
}

func NewDocumentsHandler(service *document.Service) *DocumentsHandler {
	return &DocumentsHandler{service: service}
}

func (handler *DocumentsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/projects/{project}/documents", handler.get)
	mux.HandleFunc("POST /v1/projects/{project}/documents", handler.declare)
	mux.HandleFunc("POST /v1/projects/{project}/documents/withdraw", handler.withdraw)
	mux.HandleFunc("POST /v1/projects/{project}/context-pack/rebuild", handler.rebuild)
	mux.HandleFunc("GET /v1/projects/{project}/context-pack/{pack}", handler.pack)
	mux.HandleFunc("/v1/projects/{project}/documents", methodNotAllowed("GET, POST"))
}

func (handler *DocumentsHandler) get(response http.ResponseWriter, request *http.Request) {
	view, err := handler.service.Get(request.PathValue("project"))
	handler.respond(response, view, err)
}

func (handler *DocumentsHandler) declare(response http.ResponseWriter, request *http.Request) {
	var body document.DeclareRequest
	if !decodeDocumentBody(response, request, &body, &body.Actor) {
		return
	}
	view, err := handler.service.Declare(request.PathValue("project"), body)
	handler.respond(response, view, err)
}

func (handler *DocumentsHandler) withdraw(response http.ResponseWriter, request *http.Request) {
	var body document.WithdrawRequest
	if !decodeDocumentBody(response, request, &body, &body.Actor) {
		return
	}
	view, err := handler.service.Withdraw(request.PathValue("project"), body)
	handler.respond(response, view, err)
}

func (handler *DocumentsHandler) rebuild(response http.ResponseWriter, request *http.Request) {
	var body document.RebuildRequest
	if !decodeDocumentBody(response, request, &body, &body.Actor) {
		return
	}
	view, err := handler.service.Rebuild(request.PathValue("project"), body)
	handler.respond(response, view, err)
}

func (handler *DocumentsHandler) pack(response http.ResponseWriter, request *http.Request) {
	pack, err := handler.service.Pack(request.PathValue("project"), request.PathValue("pack"))
	if err != nil {
		writeDocumentError(response, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, pack)
}

func (handler *DocumentsHandler) respond(response http.ResponseWriter, view *document.View, err error) {
	if err != nil {
		writeDocumentError(response, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, view)
}

func decodeDocumentBody(response http.ResponseWriter, request *http.Request, body any, actor *string) bool {
	if ct := request.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeError(response, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Document mutations accept application/json bodies only.")
		return false
	}
	decoder := json.NewDecoder(io.LimitReader(request.Body, maxMutationBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(body); err != nil {
		writeError(response, http.StatusBadRequest, "INVALID_MUTATION_BODY", "The mutation body could not be parsed: "+err.Error())
		return false
	}
	if *actor == "" {
		*actor = strings.TrimSpace(request.Header.Get(ActorHeader))
	}
	return true
}

func writeDocumentError(response http.ResponseWriter, err error) {
	var typed *document.Error
	if !errors.As(err, &typed) {
		writeChangeError(response, err)
		return
	}
	status := http.StatusUnprocessableEntity
	switch typed.Code {
	case document.CodeProjectNotFound, document.CodePackNotFound, document.CodeNotDeclared:
		status = http.StatusNotFound
	case document.CodeVersionConflict, document.CodeAlreadyDeclared, document.CodeManifestProtected:
		status = http.StatusConflict
	case document.CodeStateUnavailable:
		status = http.StatusServiceUnavailable
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(struct {
		Kind    string `json:"kind"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Kind: "DocumentBaselineError", Code: typed.Code, Message: typed.Message})
}
