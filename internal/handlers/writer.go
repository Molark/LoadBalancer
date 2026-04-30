package handlers

import (
	"booking/internal/repository"
	"booking/internal/service"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if data == nil {
		slog.Error("RespondJSON: data is nil")
		w.WriteHeader(status)
		return
	}
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response",
			slog.Any("error", err),
		)
		RespondError(w, ErrInternal)
		return
	}

}

func RespondError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	statusCode, errorCode, message := MapErrorToHTTP(err)
	errResp := ErrorResponse{
		Error: ErrorDetail{
			Code:    errorCode,
			Message: message,
		},
	}
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Error("Failed to encode error response",
			slog.Any("error", err),
		)
		return
	}

}

func MapErrorToHTTP(err error) (statusCode int, errorCode string, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	switch {
	// 400
	case errors.Is(err, ErrInvalidRequest):
		return http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	// 401
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED", "not authorized"
	// 403
	case errors.Is(err, service.ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN", "access denied"
	// 404
	case errors.Is(err, repository.ErrNotFound),
		errors.Is(err, repository.ErrBookingNotFound):
		return http.StatusNotFound, "NOT_FOUND", "resource not found"

	case errors.Is(err, repository.ErrRoomNotFound):
		return http.StatusNotFound, "ROOM_NOT_FOUND", "room not found"

	case errors.Is(err, repository.ErrSlotNotFound):
		return http.StatusNotFound, "SLOT_NOT_FOUND", "slot not found"

	//403
	case errors.Is(err, repository.ErrSlotAlreadyBooked):
		return http.StatusForbidden, "SLOT_ALREADY_BOOKED", "slot already booked"

	case errors.Is(err, repository.ErrScheduleExists):
		return http.StatusConflict, "SCHEDULE_EXISTS", "schedule for this room already exists and cannot be changed"

	// 500
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"
	}
}
