package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/riskpoint-per/hotel-booking/model"
)

type HotelService interface {
	FindByName(ctx context.Context, name string) ([]model.Hotel, error)
	Get(ctx context.Context, id int64) (model.Hotel, error)
	FindAvailableRooms(ctx context.Context, q model.AvailabilityQuery) ([]model.Room, error)
}

type HotelHandler struct {
	svc HotelService
}

func NewHotelHandler(svc HotelService) *HotelHandler {
	return &HotelHandler{svc: svc}
}

// List handles GET /hotels?name={name}
func (h *HotelHandler) List(w http.ResponseWriter, r *http.Request) {
	hotels, err := h.svc.FindByName(r.Context(), r.URL.Query().Get("name"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": hotels, "count": len(hotels)})
}

// Get handles GET /hotels/{id}
func (h *HotelHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	hotel, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, hotel)
}

// AvailableRooms handles
// GET /hotels/{id}/rooms/available?checkIn=YYYY-MM-DD&checkOut=YYYY-MM-DD&guests=N
func (h *HotelHandler) AvailableRooms(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	query := model.AvailabilityQuery{HotelID: id}
	invalid := map[string]string{}

	var err error
	if query.CheckIn, err = model.ParseDate(q.Get("checkIn")); err != nil {
		invalid["checkIn"] = "is required in YYYY-MM-DD format"
	}
	if query.CheckOut, err = model.ParseDate(q.Get("checkOut")); err != nil {
		invalid["checkOut"] = "is required in YYYY-MM-DD format"
	}
	if query.Guests, err = strconv.Atoi(q.Get("guests")); err != nil {
		invalid["guests"] = "is required and must be a whole number"
	}
	if len(invalid) > 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "the request is invalid", invalid)
		return
	}

	rooms, err := h.svc.FindAvailableRooms(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"hotelId":  id,
		"checkIn":  query.CheckIn,
		"checkOut": query.CheckOut,
		"guests":   query.Guests,
		"data":     rooms,
		"count":    len(rooms),
	})
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "id must be a positive integer", nil)
		return 0, false
	}
	return id, true
}
