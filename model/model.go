package model

import "time"

type RoomType string

const (
	RoomTypeSingle RoomType = "single"
	RoomTypeDouble RoomType = "double"
	RoomTypeDeluxe RoomType = "deluxe"
)

type Hotel struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"createdAt"`
}

type Room struct {
	ID       int64    `json:"id"`
	HotelID  int64    `json:"hotelId"`
	Number   string   `json:"roomNumber"`
	Type     RoomType `json:"roomType"`
	Capacity int      `json:"capacity"`
}

type Booking struct {
	ID         int64     `json:"-"`
	Reference  string    `json:"reference"`
	HotelID    int64     `json:"hotelId"`
	HotelName  string    `json:"hotelName"`
	RoomID     int64     `json:"roomId"`
	RoomNumber string    `json:"roomNumber"`
	RoomType   RoomType  `json:"roomType"`
	GuestName  string    `json:"guestName"`
	GuestEmail string    `json:"guestEmail"`
	Guests     int       `json:"guests"`
	CheckIn    Date      `json:"checkIn"`
	CheckOut   Date      `json:"checkOut"`
	Nights     int       `json:"nights"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CreateBookingRequest is the input for booking a room.
type CreateBookingRequest struct {
	HotelID    int64  `json:"hotelId"`
	RoomID     int64  `json:"roomId"`
	GuestName  string `json:"guestName"`
	GuestEmail string `json:"guestEmail"`
	Guests     int    `json:"guests"`
	CheckIn    Date   `json:"checkIn"`
	CheckOut   Date   `json:"checkOut"`
}

// AvailabilityQuery is the input for searching available rooms
type AvailabilityQuery struct {
	HotelID  int64
	CheckIn  Date
	CheckOut Date
	Guests   int
}
