package handlers

import (
	"booking/internal/contextKeys"
	"booking/internal/models"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest = errors.New("bad request")
	ErrNotFound       = errors.New("not found")
)

func (h Handler) GetRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.service.GetRooms(r.Context())
	if err != nil {
		slog.Error("Error in request to service GetRooms")
		RespondError(w, err)
		return
	}
	type Response struct {
		Rooms []models.Room `json:"rooms"`
	}
	RespondJSON(w, http.StatusOK, Response{rooms})
}

func (h Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	room := models.Room{}
	err := ReadRequestBody(r, &room)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		return
	}
	room, err = h.service.CreateRoom(r.Context(), room)
	if err != nil {
		slog.Error("Error in request to service CreateRoom")
		RespondError(w, err)
		return
	}
	type Response struct {
		Room models.Room `json:"room"`
	}
	RespondJSON(w, http.StatusCreated, Response{room})
}
func (h Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	schedule := models.Schedule{}
	err := ReadRequestBody(r, &schedule)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Error in parsing CreateSchedule", slog.Any("err", err))
		return
	}
	schedule.RoomId, err = uuid.Parse(r.PathValue("roomId"))
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Error in parsing CreateSchedule", slog.Any("err", err))
		return
	}

	schedule, err = h.service.CreateSchedule(r.Context(), schedule)
	if err != nil {
		slog.Error("Error in request to service CreateSchedule")
		RespondError(w, err)
		return
	}
	type Response struct {
		Schedule models.Schedule `json:"schedule"`
	}
	RespondJSON(w, http.StatusCreated, Response{schedule})
}

func (h Handler) GetSlots(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("roomId"))
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		slog.Error("Error in parsing GetSlots", slog.Any("err", err))
		return
	}
	date := r.URL.Query().Get("date")
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		slog.Error("GetSlots: invalid date format")
		RespondError(w, ErrInvalidRequest)
		return
	}
	slots, err := h.service.GetSlots(r.Context(), id, parsedDate)
	if err != nil {
		RespondError(w, err)
		return
	}
	type Response struct {
		Slots []models.Slot `json:"slots"`
	}
	RespondJSON(w, http.StatusOK, Response{slots})
}

func (h Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	booking := models.Booking{}
	err := ReadRequestBody(r, &booking)
	if err != nil {
		slog.Error("Error in parsing CreateBooking", slog.Any("err", err))
		RespondError(w, ErrInvalidRequest)
		return
	}
	UserId := r.Context().Value(contextKeys.UserIdKey)
	booking.UserId = UserId.(uuid.UUID)
	booking, err = h.service.CreateBooking(r.Context(), booking)
	if err != nil {
		slog.Error("Error in request to service CreateBooking")
		RespondError(w, err)
		return
	}
	type Response struct {
		Booking models.Booking `json:"booking"`
	}
	RespondJSON(w, http.StatusCreated, Response{booking})
}

func (h Handler) GetAllBookings(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")
	if pageStr == "" {
		pageStr = "1"
	}
	if pageSizeStr == "" {
		pageSizeStr = "20"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		RespondError(w, ErrInvalidRequest)
	}
	type Response struct {
		Bookings   []models.Booking  `json:"bookings"`
		Pagination models.Pagination `json:"pagination"`
	}
	bookings, total, err := h.service.GetBookings(r.Context(), page, pageSize)
	if err != nil {
		RespondError(w, err)
		return
	}
	response := Response{
		Bookings: bookings,
		Pagination: models.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
	RespondJSON(w, http.StatusOK, response)
}

func (h Handler) GetMyBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.service.GetMyBookings(r.Context())
	if err != nil {
		RespondError(w, err)
		return
	}
	type Response struct {
		Bookings []models.Booking `json:"bookings"`
	}
	RespondJSON(w, http.StatusOK, Response{bookings})
}

func (h Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	bookingId, err := uuid.Parse(r.PathValue("bookingId"))
	if err != nil {
		RespondError(w, ErrInvalidRequest)
		return
	}
	booking, err := h.service.CancelBooking(r.Context(), bookingId)
	if err != nil {
		slog.Error("Error in request to service CancelBooking", slog.Any("err", err))
		RespondError(w, err)
		return
	}
	type Response struct {
		Booking models.Booking `json:"booking"`
	}
	RespondJSON(w, http.StatusOK, Response{booking})
}

func (h Handler) GetInfo(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, "Service alive")
	slog.Info("Service is alive")
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			slog.Info("CORS : ", slog.Any("origin", origin))
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "localhost")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Origin")

		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
