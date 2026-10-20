package server

import (
	"log/slog"
	"net/http"

	"github.com/riskpoint-per/hotel-booking/handler"
)

type Handlers struct {
	Hotels   *handler.HotelHandler
	Bookings *handler.BookingHandler
	Admin    *handler.AdminHandler
}

func NewRouter(h Handlers, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Admin.Health)

	mux.HandleFunc("GET /hotels", h.Hotels.List)
	mux.HandleFunc("GET /hotels/{id}", h.Hotels.Get)
	mux.HandleFunc("GET /hotels/{id}/rooms/available", h.Hotels.AvailableRooms)

	mux.HandleFunc("POST /bookings", h.Bookings.Create)
	mux.HandleFunc("GET /bookings/{reference}", h.Bookings.Get)

	mux.HandleFunc("POST /admin/seed", h.Admin.Seed)
	mux.HandleFunc("POST /admin/reset", h.Admin.Reset)

	return recoverPanics(logger, logRequests(logger, mux))
}
