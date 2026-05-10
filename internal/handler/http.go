package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"emptyclassroom/internal/config"
	"emptyclassroom/internal/model"
	"emptyclassroom/internal/service"
)

type Handler struct {
	svc            *service.ClassroomService
	allowedOrigins []string
	syncLimiter    *rateLimiter
}

func New(svc *service.ClassroomService, cfg config.Config) *Handler {
	return &Handler{
		svc:            svc,
		allowedOrigins: cfg.AllowedOrigins,
		syncLimiter:    newRateLimiter(1, 5*time.Minute),
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", method(http.MethodGet, h.health))
	mux.HandleFunc("/api/campuses", method(http.MethodGet, h.campuses))
	mux.HandleFunc("/api/slots", method(http.MethodGet, h.slots))
	mux.HandleFunc("/api/classrooms", method(http.MethodGet, h.classrooms))
	mux.HandleFunc("/api/sync", method(http.MethodPost, h.sync))
	return h.cors(h.log(mux))
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) campuses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Campuses())
}

func (h *Handler) slots(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Slots())
}

func (h *Handler) classrooms(w http.ResponseWriter, r *http.Request) {
	campusID := parseInt(r.URL.Query().Get("campusId"), 0)
	slots, err := parseSlots(r.URL.Query().Get("slots"), r.URL.Query().Get("slot"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	date, err := parseDate(r.URL.Query().Get("date"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	summary, err := h.svc.List(r.Context(), campusID, date, slots)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) sync(w http.ResponseWriter, r *http.Request) {
	if !h.syncLimiter.Allow() {
		writeError(w, http.StatusTooManyRequests, "sync rate limited, try again later")
		return
	}

	campusText := r.URL.Query().Get("campusId")
	var err error
	if campusText == "" {
		err = h.svc.SyncAllToday(r.Context(), []int{0, 1})
	} else {
		err = h.svc.SyncCampusToday(r.Context(), parseInt(campusText, 0))
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
}

func (h *Handler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && h.originAllowed(origin) {
			if len(h.allowedOrigins) == 1 && h.allowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) originAllowed(origin string) bool {
	for _, allowed := range h.allowedOrigins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

func (h *Handler) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", value)
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseSlots(slotsParam, slotParam string) ([]int, error) {
	raw := slotsParam
	if raw == "" {
		raw = slotParam
	}
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	slots := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 1 || parsed > model.SlotCount {
			return nil, fmt.Errorf("slot must be between 1 and %d", model.SlotCount)
		}
		slots = append(slots, parsed)
	}
	return slots, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func method(allowed string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != allowed {
			w.Header().Set("Allow", allowed)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		next(w, r)
	}
}

type rateLimiter struct {
	mu       sync.Mutex
	tokens   int
	max      int
	interval time.Duration
	last     time.Time
}

func newRateLimiter(max int, interval time.Duration) *rateLimiter {
	return &rateLimiter{
		tokens:   max,
		max:      max,
		interval: interval,
		last:     time.Now(),
	}
}

func (rl *rateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.last)
	refill := int(elapsed / rl.interval)
	if refill > 0 {
		rl.tokens += refill
		if rl.tokens > rl.max {
			rl.tokens = rl.max
		}
		rl.last = rl.last.Add(time.Duration(refill) * rl.interval)
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}
