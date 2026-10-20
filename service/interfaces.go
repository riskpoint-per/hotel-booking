package service

import (
	"context"

	"github.com/riskpoint-per/hotel-booking/model"
)

type HotelStore interface {
	FindByName(ctx context.Context, name string) ([]model.Hotel, error)
	GetByID(ctx context.Context, id int64) (model.Hotel, error)
}

type RoomStore interface {
	GetByID(ctx context.Context, id int64) (model.Room, error)
	FindAvailable(ctx context.Context, q model.AvailabilityQuery) ([]model.Room, error)
}

type BookingStore interface {
	Create(ctx context.Context, b *model.Booking) error
	GetByReference(ctx context.Context, ref string) (model.Booking, error)
}

// Notifier sends booking confirmations. Implementations must not block
type Notifier interface {
	SendBookingConfirmation(b model.Booking)
}
