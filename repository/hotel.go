package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/riskpoint-per/hotel-booking/model"
)

type HotelRepository struct {
	db *sql.DB
}

func NewHotelRepository(db *sql.DB) *HotelRepository {
	return &HotelRepository{db: db}
}

// FindByName returns hotels whose name contains the search term
// An empty term returns every hotel
func (r *HotelRepository) FindByName(ctx context.Context, name string) ([]model.Hotel, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, address, created_at
		FROM hotels
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY name`, escapeLike(name))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hotels := []model.Hotel{}
	for rows.Next() {
		var h model.Hotel
		if err := rows.Scan(&h.ID, &h.Name, &h.Address, &h.CreatedAt); err != nil {
			return nil, err
		}
		hotels = append(hotels, h)
	}
	return hotels, rows.Err()
}

func (r *HotelRepository) GetByID(ctx context.Context, id int64) (model.Hotel, error) {
	var h model.Hotel
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, address, created_at FROM hotels WHERE id = $1`, id,
	).Scan(&h.ID, &h.Name, &h.Address, &h.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return h, model.ErrNotFound
	}
	return h, err
}

// escapeLike stops user input such as "%" or "_" from being interpreted as wildcards in a LIKE query
// It also escapes backslashes
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
