package service

import (
	"net/mail"
	"strings"

	"github.com/riskpoint-per/hotel-booking/model"
)

// MaxStayNights maximum number of nights for a single booking
const MaxStayNights = 30

// validateStay checks rules by availability searches and bookings
func validateStay(checkIn, checkOut model.Date, guests int) error {
	v := &model.ValidationError{}

	if checkIn.IsZero() {
		v.Add("checkIn", "is required")
	}
	if checkOut.IsZero() {
		v.Add("checkOut", "is required")
	}
	if !checkIn.IsZero() && checkIn.Before(model.Today().Time) {
		v.Add("checkIn", "cannot be in the past")
	}
	if !checkIn.IsZero() && !checkOut.IsZero() {
		nights := checkIn.NightsUntil(checkOut)
		switch {
		case nights < 1:
			v.Add("checkOut", "must be at least one day after checkIn")
		case nights > MaxStayNights:
			v.Add("checkOut", "stay cannot exceed 30 nights")
		}
	}
	if guests < 1 {
		v.Add("guests", "must be at least 1")
	}
	return v.ErrorOrNil()
}

// validateBookingRequest checks the request for a new booking
func validateBookingRequest(req model.CreateBookingRequest) error {
	v := &model.ValidationError{}

	if stayErr, ok := validateStay(req.CheckIn, req.CheckOut, req.Guests).(*model.ValidationError); ok {
		v.Fields = stayErr.Fields
	}
	if req.HotelID <= 0 {
		v.Add("hotelId", "is required")
	}
	if req.RoomID <= 0 {
		v.Add("roomId", "is required")
	}
	if strings.TrimSpace(req.GuestName) == "" {
		v.Add("guestName", "is required")
	}
	if _, err := mail.ParseAddress(req.GuestEmail); err != nil {
		v.Add("guestEmail", "must be a valid email address")
	}
	return v.ErrorOrNil()
}
