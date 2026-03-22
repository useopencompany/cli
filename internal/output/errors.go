package output

import (
	"encoding/json"
	"errors"
	"io"

	"go.agentprotocol.cloud/cli/internal/controlplane"
)

// errorJSON is the JSON structure for error responses.
type errorJSON struct {
	Error  string `json:"error"`
	Code   string `json:"code,omitempty"`
	Status int    `json:"status,omitempty"`
}

// ErrorJSON writes a JSON-formatted error to w with a trailing newline.
// If err wraps a controlplane.APIError, the code and status fields are included.
func ErrorJSON(w io.Writer, err error) {
	out := errorJSON{Error: err.Error()}

	var apiErr *controlplane.APIError
	if errors.As(err, &apiErr) {
		out.Error = apiErr.Msg
		out.Code = apiErr.Code
		out.Status = apiErr.Status
	}

	data, _ := json.Marshal(out)
	data = append(data, '\n')
	w.Write(data) //nolint:errcheck
}
