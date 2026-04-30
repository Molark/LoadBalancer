package handlers

import (
	"booking/internal/auth"
	"booking/internal/service"
	"log/slog"
	"net/http"
)

type Handler struct {
	service service.Service
	auth    *auth.AuthService
}

func NewHandler(service service.Service, auth *auth.AuthService) *Handler {
	slog.Info("NewHandler created")
	return &Handler{service: service, auth: auth}
}
func (h Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /dummyLogin", h.dummyLogin)
	mux.HandleFunc("GET /_info", h.GetInfo)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /login", h.Login)

	mux.Handle("GET /rooms/list", h.JWTAuthMiddleware(http.HandlerFunc(h.GetRooms)))
	mux.Handle("POST /rooms/create", h.JWTAuthMiddleware(http.HandlerFunc(h.CreateRoom)))
	mux.Handle("POST /rooms/{roomId}/schedule/create", h.JWTAuthMiddleware(http.HandlerFunc(h.CreateSchedule)))
	mux.Handle("GET /rooms/{roomId}/slots/list", h.JWTAuthMiddleware(http.HandlerFunc(h.GetSlots)))
	mux.Handle("POST /bookings/create", h.JWTAuthMiddleware(http.HandlerFunc(h.CreateBooking)))
	mux.Handle("GET /bookings/list", h.JWTAuthMiddleware(http.HandlerFunc(h.GetAllBookings)))
	mux.Handle("GET /bookings/my", h.JWTAuthMiddleware(http.HandlerFunc(h.GetMyBookings)))
	mux.Handle("POST /bookings/{bookingId}/cancel", h.JWTAuthMiddleware(http.HandlerFunc(h.CancelBooking)))
	slog.Info("Routers ready")
}
