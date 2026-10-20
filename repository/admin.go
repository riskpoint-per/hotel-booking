package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/riskpoint-per/hotel-booking/model"
)

// SeedBookingReference is the reference of the booking created by Seed
const SeedBookingReference = "BK-SEED0001"

// SeedSummary reports what Seed inserted.
type SeedSummary struct {
	RoomTypes        int    `json:"roomTypes"`
	Hotels           int    `json:"hotels"`
	Rooms            int    `json:"rooms"`
	Bookings         int    `json:"bookings"`
	BookingReference string `json:"sampleBookingReference"`
}

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

const truncateAll = `TRUNCATE bookings, rooms, hotels, room_types RESTART IDENTITY CASCADE`

// Reset removes all data, leaving the schema in place ready for seeding
func (r *AdminRepository) Reset(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, truncateAll)
	return err
}

// Seed resets the database and inserts a sample data:
// 3 room types, 2 hotels with 6 rooms each (2 single, 2 double, 2 deluxe)
// and one sample booking
func (r *AdminRepository) Seed(ctx context.Context) (SeedSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return SeedSummary{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, truncateAll); err != nil {
		return SeedSummary{}, fmt.Errorf("reset before seed: %w", err)
	}

	roomTypes := []struct {
		name     model.RoomType
		capacity int
	}{
		{model.RoomTypeSingle, 1},
		{model.RoomTypeDouble, 2},
		{model.RoomTypeDeluxe, 4},
	}
	for _, rt := range roomTypes {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO room_types (name, capacity) VALUES ($1, $2)`, rt.name, rt.capacity); err != nil {
			return SeedSummary{}, fmt.Errorf("seed room type %s: %w", rt.name, err)
		}
	}

	hotels := []struct{ name, address string }{
		{"Seaside Retreat", "1 Ocean Drive, Brighton"},
		{"City Central Hotel", "42 High Street, London"},
	}
	rooms := []struct {
		number string
		kind   model.RoomType
	}{
		{"101", model.RoomTypeSingle}, {"102", model.RoomTypeSingle},
		{"201", model.RoomTypeDouble}, {"202", model.RoomTypeDouble},
		{"301", model.RoomTypeDeluxe}, {"302", model.RoomTypeDeluxe},
	}

	summary := SeedSummary{RoomTypes: len(roomTypes), BookingReference: SeedBookingReference}
	var sampleRoomID int64

	for _, h := range hotels {
		var hotelID int64
		if err := tx.QueryRowContext(ctx,
			`INSERT INTO hotels (name, address) VALUES ($1, $2) RETURNING id`, h.name, h.address,
		).Scan(&hotelID); err != nil {
			return SeedSummary{}, fmt.Errorf("seed hotel %s: %w", h.name, err)
		}
		summary.Hotels++

		for _, room := range rooms {
			var roomID int64
			if err := tx.QueryRowContext(ctx,
				`INSERT INTO rooms (hotel_id, room_number, room_type) VALUES ($1, $2, $3) RETURNING id`,
				hotelID, room.number, room.kind,
			).Scan(&roomID); err != nil {
				return SeedSummary{}, fmt.Errorf("seed room %s: %w", room.number, err)
			}
			summary.Rooms++
			if sampleRoomID == 0 && room.kind == model.RoomTypeDouble {
				sampleRoomID = roomID
			}
		}
	}

	// A booking a week from today, so it is always in the future
	checkIn := model.Today().AddDays(7)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO bookings (reference, room_id, guest_name, guest_email, guests, check_in, check_out)
		VALUES ($1, $2, 'Jane Doe', 'jane.doe@example.com', 2, $3, $4)`,
		SeedBookingReference, sampleRoomID, checkIn, checkIn.AddDays(3),
	); err != nil {
		return SeedSummary{}, fmt.Errorf("seed booking: %w", err)
	}
	summary.Bookings = 1

	if err := tx.Commit(); err != nil {
		return SeedSummary{}, err
	}
	return summary, nil
}
