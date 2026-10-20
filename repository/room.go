package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/riskpoint-per/hotel-booking/model"
)

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) GetByID(ctx context.Context, id int64) (model.Room, error) {
	var room model.Room
	err := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.hotel_id, r.room_number, r.room_type, rt.capacity
		FROM rooms r
		JOIN room_types rt ON rt.name = r.room_type
		WHERE r.id = $1`, id,
	).Scan(&room.ID, &room.HotelID, &room.Number, &room.Type, &room.Capacity)
	if errors.Is(err, sql.ErrNoRows) {
		return room, model.ErrNotFound
	}
	return room, err
}

// FindAvailable returns rooms in the hotel that can hold the requested number
// of guests and are free for every night of the stay. Because a room is only
// returned when it is free for the *whole* range, booking any result means
// the guest never has to change rooms
func (r *RoomRepository) FindAvailable(ctx context.Context, q model.AvailabilityQuery) ([]model.Room, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.hotel_id, r.room_number, r.room_type, rt.capacity
		FROM rooms r
		JOIN room_types rt ON rt.name = r.room_type
		WHERE r.hotel_id = $1
		  AND rt.capacity >= $2
		  AND NOT EXISTS (
		      SELECT 1 FROM bookings b
		      WHERE b.room_id = r.id
		        AND daterange(b.check_in, b.check_out, '[)') && daterange($3::date, $4::date, '[)')
		  )
		ORDER BY rt.capacity, r.room_number`,
		q.HotelID, q.Guests, q.CheckIn, q.CheckOut)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := []model.Room{}
	for rows.Next() {
		var room model.Room
		if err := rows.Scan(&room.ID, &room.HotelID, &room.Number, &room.Type, &room.Capacity); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, rows.Err()
}
