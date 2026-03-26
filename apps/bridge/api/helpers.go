package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// Pagination holds parsed page/per_page values.
type Pagination struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Offset  int `json:"-"`
}

// ParsePagination extracts page/per_page from query params with defaults.
func ParsePagination(r *http.Request) Pagination {
	p := Pagination{Page: 1, PerPage: 50}

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Page = n
		}
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			p.PerPage = n
		}
	}
	p.Offset = (p.Page - 1) * p.PerPage
	return p
}

// TimeRange holds parsed from/to date filters.
type TimeRange struct {
	From *time.Time
	To   *time.Time
}

// ParseTimeRange extracts from/to query params (RFC3339 or YYYY-MM-DD).
func ParseTimeRange(r *http.Request) TimeRange {
	var tr TimeRange
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			tr.From = &t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			tr.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			tr.To = &t
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24*time.Hour - time.Nanosecond) // end of day
			tr.To = &end
		}
	}
	return tr
}

// PagedResponse wraps a list response with pagination metadata.
type PagedResponse struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

// Meta holds pagination metadata.
type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("response write failed", "error", err)
	}
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// WritePaged writes a paginated JSON response.
func WritePaged(w http.ResponseWriter, data any, pg Pagination, total int) {
	WriteJSON(w, http.StatusOK, PagedResponse{
		Data: data,
		Meta: Meta{Page: pg.Page, PerPage: pg.PerPage, Total: total},
	})
}

// AdminAuth middleware checks the Authorization header against the admin token.
func AdminAuth(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			WriteError(w, http.StatusForbidden, "admin not configured")
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token {
			WriteError(w, http.StatusUnauthorized, "invalid admin token")
			return
		}
		next(w, r)
	}
}

// QueryInt returns an int query param with a default value.
func QueryInt(r *http.Request, key string, defaultVal int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
