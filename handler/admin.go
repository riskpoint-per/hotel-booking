package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/riskpoint-per/hotel-booking/repository"
)

type AdminStore interface {
	Seed(ctx context.Context) (repository.SeedSummary, error)
	Reset(ctx context.Context) error
}

// AdminHandler exposes test-support endpoints. In a real deployment these
// would be disabled or protected; the brief requires no authentication.
type AdminHandler struct {
	store AdminStore
	db    *sql.DB
}

func NewAdminHandler(store AdminStore, db *sql.DB) *AdminHandler {
	return &AdminHandler{store: store, db: db}
}

// Seed handles POST /admin/seed
func (h *AdminHandler) Seed(w http.ResponseWriter, r *http.Request) {
	summary, err := h.store.Seed(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": "database seeded", "seeded": summary})
}

// Reset handles POST /admin/reset
func (h *AdminHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Reset(r.Context()); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "all data removed"})
}

// Health handles GET /health and reports database connectivity.
func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy", "database": "unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "ok"})
}
