package registryhttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"context-rail/internal/release"
)

// ReleasesHandler exposes staging promotion (VS-006).
//
//	GET  /v1/projects/{project}/releases
//	POST /v1/projects/{project}/releases                       bundle accepted candidates → promotion gate
//	GET  /v1/projects/{project}/releases/{release}
//	POST /v1/projects/{project}/releases/{release}/build       record the image build (digest identity)
//	POST /v1/projects/{project}/releases/{release}/approval    third human decision, bound to the manifest hash
//	POST /v1/projects/{project}/releases/{release}/deployment  record what was deployed → drift checks → receipt
type ReleasesHandler struct {
	service *release.Service
}

func NewReleasesHandler(service *release.Service) *ReleasesHandler {
	return &ReleasesHandler{service: service}
}

func (handler *ReleasesHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/projects/{project}/releases", handler.list)
	mux.HandleFunc("POST /v1/projects/{project}/releases", handler.create)
	mux.HandleFunc("GET /v1/projects/{project}/releases/{release}", handler.get)
	mux.HandleFunc("POST /v1/projects/{project}/releases/{release}/build", handler.build)
	mux.HandleFunc("POST /v1/projects/{project}/releases/{release}/approval", handler.approval)
	mux.HandleFunc("POST /v1/projects/{project}/releases/{release}/deployment", handler.deployment)
	mux.HandleFunc("/v1/projects/{project}/releases", methodNotAllowed("GET, POST"))
	mux.HandleFunc("/v1/projects/{project}/releases/{release}", methodNotAllowed("GET"))
}

type releaseList struct {
	Kind          string         `json:"kind"`
	SchemaVersion string         `json:"schema_version"`
	ProjectID     string         `json:"project_id"`
	Releases      []release.View `json:"releases"`
}

func (handler *ReleasesHandler) list(response http.ResponseWriter, request *http.Request) {
	projectID := request.PathValue("project")
	views, err := handler.service.List(projectID)
	if err != nil {
		writeReleaseError(response, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, releaseList{Kind: "ReleaseList", SchemaVersion: release.SchemaVersion, ProjectID: projectID, Releases: views})
}

func (handler *ReleasesHandler) get(response http.ResponseWriter, request *http.Request) {
	view, err := handler.service.Get(request.PathValue("project"), request.PathValue("release"))
	handler.respond(response, view, err)
}

func (handler *ReleasesHandler) create(response http.ResponseWriter, request *http.Request) {
	var body release.CreateRequest
	if !decodeReleaseBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.Create(request.PathValue("project"), body)
	handler.respond(response, view, err)
}

func (handler *ReleasesHandler) build(response http.ResponseWriter, request *http.Request) {
	var body release.BuildRequest
	if !decodeReleaseBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.RecordBuild(request.PathValue("project"), request.PathValue("release"), body)
	handler.respond(response, view, err)
}

func (handler *ReleasesHandler) approval(response http.ResponseWriter, request *http.Request) {
	var body release.ApprovalRequest
	if !decodeReleaseBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.Approve(request.PathValue("project"), request.PathValue("release"), body)
	handler.respond(response, view, err)
}

func (handler *ReleasesHandler) deployment(response http.ResponseWriter, request *http.Request) {
	var body release.DeploymentRequest
	if !decodeReleaseBody(response, request, &body, &body.Mutation) {
		return
	}
	view, err := handler.service.RecordDeployment(request.PathValue("project"), request.PathValue("release"), body)
	handler.respond(response, view, err)
}

func (handler *ReleasesHandler) respond(response http.ResponseWriter, view *release.View, err error) {
	if err != nil {
		writeReleaseError(response, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, view)
}

func decodeReleaseBody(response http.ResponseWriter, request *http.Request, body any, mutation *release.Mutation) bool {
	if ct := request.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		writeError(response, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Release mutations accept application/json bodies only.")
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

func writeReleaseError(response http.ResponseWriter, err error) {
	var typed *release.Error
	if !errors.As(err, &typed) {
		writeChangeError(response, err)
		return
	}
	status := http.StatusUnprocessableEntity
	switch typed.Code {
	case release.CodeProjectNotFound, release.CodeReleaseNotFound:
		status = http.StatusNotFound
	case release.CodeAlreadyApproved, release.CodeAlreadyPromoted, release.CodeReleaseClosed, release.CodeStale, release.CodeGateBlocked, release.CodeNotApprovable, release.CodeApproverConflict, release.CodeBuildMismatch:
		status = http.StatusConflict
	case release.CodeStateUnavailable:
		status = http.StatusServiceUnavailable
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(struct {
		Kind    string `json:"kind"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Kind: "ReleaseError", Code: typed.Code, Message: typed.Message})
}
