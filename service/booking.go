package service

import (
	"context"
	"errors"
	"strings"

	"github.com/riskpoint-per/hotel-booking/model"
)

const maxReferenceAttempts = 5

type BookingService struct {
	hotels   HotelStore
	rooms    RoomStore
	bookings BookingStore
	notifier Notifier
	newRef   func() (string, error)
}

func NewBookingService(h HotelStore, r RoomStore, b BookingStore, n Notifier) *BookingService {
	return &BookingService{hotels: h, rooms: r, bookings: b, notifier: n, newRef: NewReference}
}

// Book validates the request, enforces the hotel rules and persists the
// booking. On success it triggers the confirmation email asynchronously and
// returns immediately — the caller never waits for the email
func (s *BookingService) Book(ctx context.Context, req model.CreateBookingRequest) (model.Booking, error) {
	req.GuestName = strings.TrimSpace(req.GuestName)
	req.GuestEmail = strings.TrimSpace(req.GuestEmail)

	if err := validateBookingRequest(req); err != nil {
		return model.Booking{}, err
	}

	hotel, err := s.hotels.GetByID(ctx, req.HotelID)
	if err != nil {
		return model.Booking{}, err
	}
	room, err := s.rooms.GetByID(ctx, req.RoomID)
	if err != nil {
		return model.Booking{}, err
	}
	if room.HotelID != hotel.ID {
		return model.Booking{}, model.ErrNotFound
	}
	if req.Guests > room.Capacity {
		return model.Booking{}, model.ErrCapacityExceeded
	}

	booking := model.Booking{
		HotelID:    hotel.ID,
		HotelName:  hotel.Name,
		RoomID:     room.ID,
		RoomNumber: room.Number,
		RoomType:   room.Type,
		GuestName:  req.GuestName,
		GuestEmail: req.GuestEmail,
		Guests:     req.Guests,
		CheckIn:    req.CheckIn,
		CheckOut:   req.CheckOut,
		Nights:     req.CheckIn.NightsUntil(req.CheckOut),
	}

	// Retry only on the reference duplication and until maxReferenceAttempts is reached
	// err will be returned if the booking creation fails for any other reason or if the max attempts is reached
	for attempt := 1; ; attempt++ {
		booking.Reference, err = s.newRef()
		if err != nil {
			return model.Booking{}, err
		}
		err = s.bookings.Create(ctx, &booking)
		if !errors.Is(err, model.ErrDuplicateReference) || attempt == maxReferenceAttempts {
			break
		}
	}
	if err != nil {
		return model.Booking{}, err
	}

	s.notifier.SendBookingConfirmation(booking)
	return booking, nil
}

func (s *BookingService) GetByReference(ctx context.Context, ref string) (model.Booking, error) {
	ref = strings.ToUpper(strings.TrimSpace(ref))
	if ref == "" {
		return model.Booking{}, model.ErrNotFound
	}
	return s.bookings.GetByReference(ctx, ref)
}
