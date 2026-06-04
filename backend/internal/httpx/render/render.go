package render

import (
	"context"
	"encoding/json"
	"net/http"

	appmiddleware "github.com/AsperforMias/ScriptForge/backend/internal/httpx/middleware"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

type errorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type responseEnvelope struct {
	Data  any        `json:"data"`
	Error *errorBody `json:"error"`
	Meta  metaBody   `json:"meta"`
}

type metaBody struct {
	RequestID string `json:"request_id"`
}

func WriteJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	writeEnvelope(ctx, w, status, responseEnvelope{
		Data:  data,
		Error: nil,
		Meta: metaBody{
			RequestID: appmiddleware.RequestIDFromContext(ctx),
		},
	})
}

func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	status, body := mapError(err)
	writeEnvelope(ctx, w, status, responseEnvelope{
		Data: nil,
		Error: &errorBody{
			Code:    body.Code,
			Message: body.Message,
			Details: body.Details,
		},
		Meta: metaBody{
			RequestID: appmiddleware.RequestIDFromContext(ctx),
		},
	})
}

func writeEnvelope(_ context.Context, w http.ResponseWriter, status int, envelope responseEnvelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope)
}

func mapError(err error) (int, job.AppError) {
	if err == nil {
		return http.StatusInternalServerError, job.NewAppError("internal_error", "internal server error", nil)
	}

	var appErr job.AppError
	if ok := job.AsAppError(err, &appErr); ok {
		switch appErr.Code {
		case "invalid_input":
			return http.StatusBadRequest, appErr
		case "job_not_found":
			return http.StatusNotFound, appErr
		case "job_not_ready":
			return http.StatusConflict, appErr
		case "generation_failed":
			return http.StatusInternalServerError, appErr
		default:
			return http.StatusInternalServerError, appErr
		}
	}

	return http.StatusInternalServerError, job.NewAppError("internal_error", "internal server error", nil)
}
