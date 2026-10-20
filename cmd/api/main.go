// Command api starts the hotel booking HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riskpoint-per/hotel-booking/config"
	"github.com/riskpoint-per/hotel-booking/database"
	"github.com/riskpoint-per/hotel-booking/handler"
	"github.com/riskpoint-per/hotel-booking/notification"
	"github.com/riskpoint-per/hotel-booking/repository"
	"github.com/riskpoint-per/hotel-booking/server"
	"github.com/riskpoint-per/hotel-booking/service"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL, 30)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		return err
	}

	// Dependency wiring: repositories -> services -> handlers -> router.
	hotelRepo := repository.NewHotelRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	adminRepo := repository.NewAdminRepository(db)

	emailer := notification.NewEmailNotifier(cfg.EmailDelay, logger)

	hotelSvc := service.NewHotelService(hotelRepo, roomRepo)
	bookingSvc := service.NewBookingService(hotelRepo, roomRepo, bookingRepo, emailer)

	router := server.NewRouter(server.Handlers{
		Hotels:   handler.NewHotelHandler(hotelSvc),
		Bookings: handler.NewBookingHandler(bookingSvc),
		Admin:    handler.NewAdminHandler(adminRepo, db),
	}, logger)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Stop accepting requests first, then let queued emails finish.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	emailer.Shutdown(shutdownCtx)
	logger.Info("shutdown complete")
	return nil
}
