package handler

import (
	"context"
	"net/http"

	"github.com/riskpoint-per/hotel-booking/model"
)

type BookingService interface {
	Book(ctx context.Context, req model.CreateBookingRequest) (model.Booking, error)
	GetByReference(ctx context.Context, ref string) (model.Booking, error)
}

type BookingHandler struct {
	svc BookingService
}

func NewBookingHandler(svc BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

// Create handles POST /bookings
func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateBookingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}

	booking, err := h.svc.Book(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Location", "/bookings/"+booking.Reference)
	writeJSON(w, http.StatusCreated, booking)
}

// Get handles GET /bookings/{reference}
func (h *BookingHandler) Get(w http.ResponseWriter, r *http.Request) {
	booking, err := h.svc.GetByReference(r.Context(), r.PathValue("reference"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}
