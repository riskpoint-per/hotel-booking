package notification

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/riskpoint-per/hotel-booking/model"
)

// EmailNotifier simulates sending confirmation emails. Each email is handled
// by its own goroutine, so the HTTP request that created the booking returns
// without waiting. A WaitGroup tracks in-flight emails so the server can
// let them finish during graceful shutdown
type EmailNotifier struct {
	delay  time.Duration
	logger *slog.Logger
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewEmailNotifier(delay time.Duration, logger *slog.Logger) *EmailNotifier {
	ctx, cancel := context.WithCancel(context.Background())
	return &EmailNotifier{delay: delay, logger: logger, ctx: ctx, cancel: cancel}
}

// SendBookingConfirmation returns immediately; the email is "sent" after
// the configured delay on a background goroutine
func (n *EmailNotifier) SendBookingConfirmation(b model.Booking) {
	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		defer func() {
			// A panic in a background goroutine would crash the whole server.
			if r := recover(); r != nil {
				n.logger.Error("confirmation email panicked", "reference", b.Reference, "panic", r)
			}
		}()

		n.logger.Info("queued confirmation email", "reference", b.Reference, "to", b.GuestEmail)

		select {
		case <-time.After(n.delay): // simulate talking to an SMTP server
		case <-n.ctx.Done():
			n.logger.Warn("confirmation email cancelled by shutdown", "reference", b.Reference)
			return
		}

		n.logger.Info("confirmation email sent",
			"reference", b.Reference,
			"to", b.GuestEmail,
			"guest", b.GuestName,
			"hotel", b.HotelName,
			"room", b.RoomNumber,
			"checkIn", b.CheckIn.String(),
			"checkOut", b.CheckOut.String(),
		)
	}()
}

// Shutdown waits for in-flight emails to finish. If ctx expires first the
// remaining emails are cancelled
func (n *EmailNotifier) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		n.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		n.cancel()
		<-done
	}
}
