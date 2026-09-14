package registryhttp

import (
	"encoding/json"
	"net/http"
)

type apiError struct {
	Kind     string `json:"kind"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	ReadOnly bool   `json:"read_only"`
}

func writeError(response http.ResponseWriter, status int, code, message string) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(apiError{
		Kind: "ProjectRegistryError", Code: code, Message: message, ReadOnly: true,
	})
}
