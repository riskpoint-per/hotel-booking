package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DateLayout is the wire format for all dates in the API (ISO 8601 calendar date)
const DateLayout = "2006-01-02"

// Date is a calendar date with no time-of-day or timezone component
// It marshals to/from JSON as "YYYY-MM-DD" and maps onto a Postgres DATE
type Date struct {
	time.Time
}

// ParseDate parses a "YYYY-MM-DD" string
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", s)
	}
	return Date{t}, nil
}

// Today returns the current UTC calendar date
func Today() Date {
	y, m, d := time.Now().UTC().Date()
	return Date{time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}

func (d Date) String() string { return d.Format(DateLayout) }

// AddDays returns the date n days later (or earlier if n is negative)
func (d Date) AddDays(n int) Date { return Date{d.Time.AddDate(0, 0, n)} }

// NightsUntil returns the number of nights between d and other
func (d Date) NightsUntil(other Date) int {
	return int(other.Sub(d.Time).Hours() / 24)
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("date must be a string in YYYY-MM-DD format")
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Value implements driver.Valuer so a Date can be passed as a query argument
func (d Date) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan implements sql.Scanner so a Postgres DATE column scans into a Date
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		y, m, day := v.Date()
		d.Time = time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
		return nil
	case string:
		parsed, err := ParseDate(v)
		*d = parsed
		return err
	case []byte:
		parsed, err := ParseDate(string(v))
		*d = parsed
		return err
	default:
		return fmt.Errorf("cannot scan %T into Date", src)
	}
}
