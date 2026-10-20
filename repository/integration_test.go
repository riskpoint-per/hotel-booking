package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/riskpoint-per/hotel-booking/database"
	"github.com/riskpoint-per/hotel-booking/model"
	"github.com/riskpoint-per/hotel-booking/repository"
)

func setup(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	db, err := database.Connect(ctx, dsn, 5)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.NewAdminRepository(db).Seed(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}

func firstRoom(t *testing.T, db *sql.DB, kind model.RoomType) model.Room {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM rooms WHERE room_type = $1 ORDER BY id DESC LIMIT 1`, kind).Scan(&id); err != nil {
		t.Fatal(err)
	}
	room, err := repository.NewRoomRepository(db).GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return room
}

func booking(ref string, room model.Room, guests int, in, out model.Date) *model.Booking {
	return &model.Booking{Reference: ref, RoomID: room.ID, GuestName: "Test", GuestEmail: "t@example.com",
		Guests: guests, CheckIn: in, CheckOut: out}
}

func TestOverlappingBookingRejected(t *testing.T) {
	db := setup(t)
	repo := repository.NewBookingRepository(db)
	room := firstRoom(t, db, model.RoomTypeDeluxe)
	in := model.Today().AddDays(20)

	if err := repo.Create(context.Background(), booking("BK-TEST0001", room, 2, in, in.AddDays(3))); err != nil {
		t.Fatal(err)
	}
	// Overlaps the last night
	err := repo.Create(context.Background(), booking("BK-TEST0002", room, 2, in.AddDays(2), in.AddDays(5)))
	if !errors.Is(err, model.ErrRoomUnavailable) {
		t.Fatalf("expected ErrRoomUnavailable, got %v", err)
	}
	// Back-to-back: check in on the previous guest's check-out day
	if err := repo.Create(context.Background(), booking("BK-TEST0003", room, 2, in.AddDays(3), in.AddDays(4))); err != nil {
		t.Fatalf("back-to-back booking should succeed: %v", err)
	}
}

func TestDuplicateReferenceRejected(t *testing.T) {
	db := setup(t)
	repo := repository.NewBookingRepository(db)
	room := firstRoom(t, db, model.RoomTypeDeluxe)
	in := model.Today().AddDays(40)

	err := repo.Create(context.Background(), booking(repository.SeedBookingReference, room, 1, in, in.AddDays(1)))
	if !errors.Is(err, model.ErrDuplicateReference) {
		t.Fatalf("expected ErrDuplicateReference, got %v", err)
	}
}

func TestDatabaseEnforcesCapacity(t *testing.T) {
	db := setup(t)
	repo := repository.NewBookingRepository(db)
	room := firstRoom(t, db, model.RoomTypeSingle)
	in := model.Today().AddDays(10)

	err := repo.Create(context.Background(), booking("BK-TEST0004", room, 2, in, in.AddDays(1)))
	if !errors.Is(err, model.ErrCapacityExceeded) {
		t.Fatalf("expected ErrCapacityExceeded, got %v", err)
	}
}

func TestHotelCannotExceedSixRooms(t *testing.T) {
	db := setup(t)
	_, err := db.Exec(`INSERT INTO rooms (hotel_id, room_number, room_type) VALUES (1, '999', 'single')`)
	if err == nil {
		t.Fatal("expected inserting a 7th room to fail")
	}
}

// TestConcurrentBookingsOnlyOneWins fires many simultaneous requests for the
// same room and nights; exactly one must succeed
func TestConcurrentBookingsOnlyOneWins(t *testing.T) {
	db := setup(t)
	repo := repository.NewBookingRepository(db)
	room := firstRoom(t, db, model.RoomTypeDouble)
	in := model.Today().AddDays(60)

	const n = 20
	var wg sync.WaitGroup
	results := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results <- repo.Create(context.Background(),
				booking(fmt.Sprintf("BK-RACE%04d", i), room, 1, in, in.AddDays(2)))
		}(i)
	}
	wg.Wait()
	close(results)

	var ok, conflicts int
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, model.ErrRoomUnavailable):
			conflicts++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 || conflicts != n-1 {
		t.Fatalf("expected 1 success and %d conflicts, got %d and %d", n-1, ok, conflicts)
	}
}
