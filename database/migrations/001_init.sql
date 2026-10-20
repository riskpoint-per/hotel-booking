-- btree_gist lets us combine an equality check (room_id) with a range
-- overlap check (stay dates) in a single exclusion constraint.
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS room_types (
    name     TEXT PRIMARY KEY CHECK (name IN ('single', 'double', 'deluxe')),
    capacity INT  NOT NULL CHECK (capacity > 0)
);

CREATE TABLE IF NOT EXISTS hotels (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT        NOT NULL UNIQUE,
    address    TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rooms (
    id          BIGSERIAL PRIMARY KEY,
    hotel_id    BIGINT NOT NULL REFERENCES hotels (id) ON DELETE CASCADE,
    room_number TEXT   NOT NULL,
    room_type   TEXT   NOT NULL REFERENCES room_types (name),
    UNIQUE (hotel_id, room_number)
);

CREATE INDEX IF NOT EXISTS idx_rooms_hotel_id ON rooms (hotel_id);

-- Rule: a hotel has 6 rooms. This trigger guarantees a hotel can never
-- exceed 6, whichever code path inserts rooms.
CREATE OR REPLACE FUNCTION enforce_max_rooms_per_hotel() RETURNS TRIGGER AS $$
BEGIN
    -- Lock the hotel row so concurrent inserts for the same hotel serialise.
    PERFORM 1 FROM hotels WHERE id = NEW.hotel_id FOR UPDATE;
    IF (SELECT COUNT(*) FROM rooms WHERE hotel_id = NEW.hotel_id) >= 6 THEN
        RAISE EXCEPTION 'hotel % already has the maximum of 6 rooms', NEW.hotel_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_max_rooms_per_hotel ON rooms;
CREATE TRIGGER trg_max_rooms_per_hotel
    BEFORE INSERT ON rooms
    FOR EACH ROW EXECUTE FUNCTION enforce_max_rooms_per_hotel();

CREATE TABLE IF NOT EXISTS bookings (
    id          BIGSERIAL PRIMARY KEY,
    -- Rule: booking references are unique.
    reference   TEXT        NOT NULL UNIQUE,
    room_id     BIGINT      NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    guest_name  TEXT        NOT NULL,
    guest_email TEXT        NOT NULL,
    guests      INT         NOT NULL CHECK (guests > 0),
    check_in    DATE        NOT NULL,
    check_out   DATE        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT valid_stay CHECK (check_out > check_in),
    -- Rule: a room can never be double booked for any night.
    -- '[)' makes check-out exclusive, so one guest can check in on the day
    -- another checks out. A booking is one row for one room covering the
    -- whole stay, so a guest never has to change rooms.
    CONSTRAINT no_overlapping_bookings EXCLUDE USING gist (
        room_id WITH =,
        daterange(check_in, check_out, '[)') WITH &&
    )
);

-- Rule: a room cannot hold more guests than its capacity. The service layer
-- checks this too; the database is the last line of defence.
CREATE OR REPLACE FUNCTION enforce_room_capacity() RETURNS TRIGGER AS $$
DECLARE
    max_guests INT;
BEGIN
    SELECT rt.capacity INTO max_guests
    FROM rooms r JOIN room_types rt ON rt.name = r.room_type
    WHERE r.id = NEW.room_id;

    IF NEW.guests > max_guests THEN
        RAISE EXCEPTION 'room % holds at most % guests', NEW.room_id, max_guests
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_room_capacity ON bookings;
CREATE TRIGGER trg_room_capacity
    BEFORE INSERT OR UPDATE ON bookings
    FOR EACH ROW EXECUTE FUNCTION enforce_room_capacity();
