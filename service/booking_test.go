package service

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"testing"

	"github.com/riskpoint-per/hotel-booking/model"
)

type fakeHotels struct{ hotels map[int64]model.Hotel }

func (f *fakeHotels) FindByName(context.Context, string) ([]model.Hotel, error) { return nil, nil }
func (f *fakeHotels) GetByID(_ context.Context, id int64) (model.Hotel, error) {
	h, ok := f.hotels[id]
	if !ok {
		return h, model.ErrNotFound
	}
	return h, nil
}

type fakeRooms struct{ rooms map[int64]model.Room }

func (f *fakeRooms) GetByID(_ context.Context, id int64) (model.Room, error) {
	r, ok := f.rooms[id]
	if !ok {
		return r, model.ErrNotFound
	}
	return r, nil
}
func (f *fakeRooms) FindAvailable(context.Context, model.AvailabilityQuery) ([]model.Room, error) {
	return nil, nil
}

type fakeBookings struct {
	mu       sync.Mutex
	created  []model.Booking
	failNext []error
}

func (f *fakeBookings) Create(_ context.Context, b *model.Booking) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.failNext) > 0 {
		err := f.failNext[0]
		f.failNext = f.failNext[1:]
		return err
	}
	f.created = append(f.created, *b)
	return nil
}
func (f *fakeBookings) GetByReference(context.Context, string) (model.Booking, error) {
	return model.Booking{}, model.ErrNotFound
}

type fakeNotifier struct{ sent []model.Booking }

func (f *fakeNotifier) SendBookingConfirmation(b model.Booking) { f.sent = append(f.sent, b) }

func newTestService() (*BookingService, *fakeBookings, *fakeNotifier) {
	hotels := &fakeHotels{hotels: map[int64]model.Hotel{
		1: {ID: 1, Name: "Seaside Retreat"},
		2: {ID: 2, Name: "City Central Hotel"},
	}}
	rooms := &fakeRooms{rooms: map[int64]model.Room{
		10: {ID: 10, HotelID: 1, Number: "101", Type: model.RoomTypeSingle, Capacity: 1},
		11: {ID: 11, HotelID: 1, Number: "201", Type: model.RoomTypeDouble, Capacity: 2},
		20: {ID: 20, HotelID: 2, Number: "101", Type: model.RoomTypeSingle, Capacity: 1},
	}}
	bookings := &fakeBookings{}
	notifier := &fakeNotifier{}
	return NewBookingService(hotels, rooms, bookings, notifier), bookings, notifier
}

func validRequest() model.CreateBookingRequest {
	in := model.Today().AddDays(1)
	return model.CreateBookingRequest{
		HotelID: 1, RoomID: 11, GuestName: "Rob Pike", GuestEmail: "rob@example.com",
		Guests: 2, CheckIn: in, CheckOut: in.AddDays(3),
	}
}

func TestBook_Success(t *testing.T) {
	svc, store, notifier := newTestService()

	b, err := svc.Book(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Nights != 3 || b.RoomNumber != "201" || b.HotelName != "Seaside Retreat" {
		t.Errorf("unexpected booking: %+v", b)
	}
	if len(store.created) != 1 {
		t.Errorf("expected 1 stored booking, got %d", len(store.created))
	}
	if len(notifier.sent) != 1 {
		t.Errorf("expected confirmation to be triggered once, got %d", len(notifier.sent))
	}
}

func TestBook_ValidationErrors(t *testing.T) {
	today := model.Today()
	tests := map[string]struct {
		mutate func(*model.CreateBookingRequest)
		field  string
	}{
		"check-out before check-in": {func(r *model.CreateBookingRequest) { r.CheckOut = r.CheckIn.AddDays(-1) }, "checkOut"},
		"zero nights":               {func(r *model.CreateBookingRequest) { r.CheckOut = r.CheckIn }, "checkOut"},
		"too long":                  {func(r *model.CreateBookingRequest) { r.CheckOut = r.CheckIn.AddDays(31) }, "checkOut"},
		"past check-in":             {func(r *model.CreateBookingRequest) { r.CheckIn = today.AddDays(-2) }, "checkIn"},
		"no guests":                 {func(r *model.CreateBookingRequest) { r.Guests = 0 }, "guests"},
		"missing name":              {func(r *model.CreateBookingRequest) { r.GuestName = "   " }, "guestName"},
		"bad email":                 {func(r *model.CreateBookingRequest) { r.GuestEmail = "not-an-email" }, "guestEmail"},
		"missing room":              {func(r *model.CreateBookingRequest) { r.RoomID = 0 }, "roomId"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			svc, _, notifier := newTestService()
			req := validRequest()
			tc.mutate(&req)

			_, err := svc.Book(context.Background(), req)
			var vErr *model.ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if _, ok := vErr.Fields[tc.field]; !ok {
				t.Errorf("expected error on field %q, got %v", tc.field, vErr.Fields)
			}
			if len(notifier.sent) != 0 {
				t.Error("no confirmation should be sent for an invalid booking")
			}
		})
	}
}

func TestBook_CapacityExceeded(t *testing.T) {
	svc, _, _ := newTestService()
	req := validRequest()
	req.RoomID = 10 // single room
	req.Guests = 2

	if _, err := svc.Book(context.Background(), req); !errors.Is(err, model.ErrCapacityExceeded) {
		t.Fatalf("expected ErrCapacityExceeded, got %v", err)
	}
}

func TestBook_RoomFromAnotherHotel(t *testing.T) {
	svc, _, _ := newTestService()
	req := validRequest()
	req.RoomID = 20 // belongs to hotel 2
	req.Guests = 1

	if _, err := svc.Book(context.Background(), req); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestBook_RoomUnavailableIsNotRetried(t *testing.T) {
	svc, store, notifier := newTestService()
	store.failNext = []error{model.ErrRoomUnavailable}

	if _, err := svc.Book(context.Background(), validRequest()); !errors.Is(err, model.ErrRoomUnavailable) {
		t.Fatalf("expected ErrRoomUnavailable, got %v", err)
	}
	if len(notifier.sent) != 0 {
		t.Error("no confirmation should be sent when the room is unavailable")
	}
}

func TestBook_RetriesOnReferenceCollision(t *testing.T) {
	svc, store, _ := newTestService()
	store.failNext = []error{model.ErrDuplicateReference, model.ErrDuplicateReference}

	if _, err := svc.Book(context.Background(), validRequest()); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if len(store.created) != 1 {
		t.Errorf("expected 1 stored booking, got %d", len(store.created))
	}
}

func TestNewReference_FormatAndUniqueness(t *testing.T) {
	pattern := regexp.MustCompile(`^BK-[0-9A-HJKMNP-TV-Z]{8}$`)
	seen := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		ref, err := NewReference()
		if err != nil {
			t.Fatal(err)
		}
		if !pattern.MatchString(ref) {
			t.Fatalf("reference %q has unexpected format", ref)
		}
		if _, dup := seen[ref]; dup {
			t.Fatalf("duplicate reference generated: %s", ref)
		}
		seen[ref] = struct{}{}
	}
}
