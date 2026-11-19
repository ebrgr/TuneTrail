package adminweb

import (
	"encoding/json"
	"github.com/ebrgr/TuneTrail/internal/db"
	"io"
	"net/http"
)

// DecodeManagementJSON rejects hidden role, status and owner fields and trailing documents.
func DecodeManagementJSON[T any](r *http.Request) (T, error) {
	var value T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, db.ErrInvalidInput
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return value, db.ErrInvalidInput
	}
	return value, nil
}
