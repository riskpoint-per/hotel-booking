package service

import (
	"context"
	"strings"

	"github.com/riskpoint-per/hotel-booking/model"
)

type HotelService struct {
	hotels HotelStore
	rooms  RoomStore
}

func NewHotelService(hotels HotelStore, rooms RoomStore) *HotelService {
	return &HotelService{hotels: hotels, rooms: rooms}
}

// FindByName searches hotels by name
// it can be partial and it's case-insensitive
func (s *HotelService) FindByName(ctx context.Context, name string) ([]model.Hotel, error) {
	return s.hotels.FindByName(ctx, strings.TrimSpace(name))
}

func (s *HotelService) Get(ctx context.Context, id int64) (model.Hotel, error) {
	return s.hotels.GetByID(ctx, id)
}

// FindAvailableRooms returns rooms free for the whole stay that fit the party
func (s *HotelService) FindAvailableRooms(ctx context.Context, q model.AvailabilityQuery) ([]model.Room, error) {
	if err := validateStay(q.CheckIn, q.CheckOut, q.Guests); err != nil {
		return nil, err
	}
	if _, err := s.hotels.GetByID(ctx, q.HotelID); err != nil {
		return nil, err
	}
	return s.rooms.FindAvailable(ctx, q)
}
