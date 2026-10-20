package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/riskpoint-per/hotel-booking/model"
)

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Create inserts a booking. Double booking is prevented by the
// no_overlapping_bookings exclusion constraint, so two concurrent requests
// for the same room and overlapping nights can never both succeed
// the failed one receives model.ErrRoomUnavailable
func (r *BookingRepository) Create(ctx context.Context, b *model.Booking) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO bookings (reference, room_id, guest_name, guest_email, guests, check_in, check_out)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		b.Reference, b.RoomID, b.GuestName, b.GuestEmail, b.Guests, b.CheckIn, b.CheckOut,
	).Scan(&b.ID, &b.CreatedAt)
	return translateBookingError(err)
}

func (r *BookingRepository) GetByReference(ctx context.Context, ref string) (model.Booking, error) {
	var b model.Booking
	err := r.db.QueryRowContext(ctx, `
		SELECT b.id, b.reference, h.id, h.name, r.id, r.room_number, r.room_type,
		       b.guest_name, b.guest_email, b.guests, b.check_in, b.check_out, b.created_at
		FROM bookings b
		JOIN rooms r  ON r.id = b.room_id
		JOIN hotels h ON h.id = r.hotel_id
		WHERE b.reference = $1`, ref,
	).Scan(&b.ID, &b.Reference, &b.HotelID, &b.HotelName, &b.RoomID, &b.RoomNumber, &b.RoomType,
		&b.GuestName, &b.GuestEmail, &b.Guests, &b.CheckIn, &b.CheckOut, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return b, model.ErrNotFound
	}
	if err != nil {
		return b, err
	}
	b.Nights = b.CheckIn.NightsUntil(b.CheckOut)
	return b, nil
}
