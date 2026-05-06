package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/dsalas/games-tracker-backend/internal/httpx"
	"github.com/dsalas/games-tracker-backend/internal/models"
	"github.com/dsalas/games-tracker-backend/internal/store"
)

// SeriesHandler agrupa los endpoints de /series. La inyección por struct
// hace fácil pasar dependencias adicionales (logger, etc.) más adelante.
type SeriesHandler struct {
	Store *store.SeriesStore
}

// Routes monta los endpoints CRUD bajo el router que recibe.
func (h *SeriesHandler) Routes(r chi.Router) {
	r.Get("/series", h.List)
	r.Post("/series", h.Create)
	r.Get("/series/{id}", h.Get)
	r.Put("/series/{id}", h.Update)
	r.Delete("/series/{id}", h.Delete)
}

// listResponse es la forma del payload paginado descrita en el CLAUDE.md.
type listResponse struct {
	Data []models.Series `json:"data"`
	Meta listMeta        `json:"meta"`
}

type listMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func (h *SeriesHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, err := parsePositiveInt(q.Get("page"), 1)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "page must be a positive integer", "page")
		return
	}
	limit, err := parsePositiveInt(q.Get("limit"), 20)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "limit must be a positive integer", "limit")
		return
	}
	if limit > 100 {
		limit = 100
	}

	sort := strings.TrimSpace(q.Get("sort"))
	if sort != "" && sort != "title" && sort != "created_at" && sort != "status" {
		httpx.Error(w, http.StatusBadRequest,
			"sort must be one of: title, created_at, status", "sort")
		return
	}

	order := strings.ToLower(strings.TrimSpace(q.Get("order")))
	if order != "" && order != "asc" && order != "desc" {
		httpx.Error(w, http.StatusBadRequest, "order must be asc or desc", "order")
		return
	}

	rows, total, err := h.Store.List(r.Context(), store.ListOptions{
		Q:     q.Get("q"),
		Sort:  sort,
		Order: order,
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list series", "")
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	httpx.JSON(w, http.StatusOK, listResponse{
		Data: rows,
		Meta: listMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages},
	})
}

func (h *SeriesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	s, err := h.Store.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "series not found", "")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get series", "")
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *SeriesHandler) Create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	// En POST, title y status son obligatorios.
	if errs := validateInput(in, true); len(errs) > 0 {
		httpx.Errors(w, http.StatusBadRequest, errs)
		return
	}
	s, err := h.Store.Insert(r.Context(), in)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to create series", "")
		return
	}
	httpx.JSON(w, http.StatusCreated, s)
}

func (h *SeriesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	in, ok := decodeInput(w, r)
	if !ok {
		return
	}
	// PUT también requiere title y status (es un reemplazo, no un patch).
	if errs := validateInput(in, true); len(errs) > 0 {
		httpx.Errors(w, http.StatusBadRequest, errs)
		return
	}
	s, err := h.Store.Update(r.Context(), id, in)
	if errors.Is(err, store.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "series not found", "")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to update series", "")
		return
	}
	httpx.JSON(w, http.StatusOK, s)
}

func (h *SeriesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.Store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "series not found", "")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to delete series", "")
		return
	}
	httpx.NoContent(w)
}

// --- helpers ---

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "id must be a positive integer", "id")
		return 0, false
	}
	return id, true
}

// parsePositiveInt convierte un parámetro numérico positivo o devuelve el
// fallback si la cadena viene vacía. Vuelve con error si el valor es <=0.
func parsePositiveInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid: %q", raw)
	}
	return n, nil
}

func decodeInput(w http.ResponseWriter, r *http.Request) (models.SeriesInput, bool) {
	var in models.SeriesInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON body: "+err.Error(), "")
		return in, false
	}
	return in, true
}

// validateInput agrupa todos los errores antes de devolverlos para que el
// cliente reciba la lista completa en una sola respuesta.
func validateInput(in models.SeriesInput, requireAll bool) []httpx.FieldError {
	var errs []httpx.FieldError

	if requireAll || in.Title != nil {
		title := ""
		if in.Title != nil {
			title = strings.TrimSpace(*in.Title)
		}
		if title == "" {
			errs = append(errs, httpx.FieldError{Field: "title", Error: "title is required"})
		} else if len(title) > 255 {
			errs = append(errs, httpx.FieldError{Field: "title", Error: "title must be at most 255 characters"})
		}
	}

	if requireAll || in.Status != nil {
		status := ""
		if in.Status != nil {
			status = strings.TrimSpace(*in.Status)
		}
		if status == "" {
			errs = append(errs, httpx.FieldError{Field: "status", Error: "status is required"})
		} else if !models.IsValidStatus(status) {
			errs = append(errs, httpx.FieldError{
				Field: "status",
				Error: "status must be one of: watching, completed, dropped, pending",
			})
		}
	}

	if in.Genre != nil && len(*in.Genre) > 100 {
		errs = append(errs, httpx.FieldError{Field: "genre", Error: "genre must be at most 100 characters"})
	}

	if in.EpisodesWatched != nil && *in.EpisodesWatched < 0 {
		errs = append(errs, httpx.FieldError{Field: "episodes_watched", Error: "episodes_watched must be >= 0"})
	}
	if in.TotalEpisodes != nil && *in.TotalEpisodes < 0 {
		errs = append(errs, httpx.FieldError{Field: "total_episodes", Error: "total_episodes must be >= 0"})
	}
	if in.EpisodesWatched != nil && in.TotalEpisodes != nil &&
		*in.TotalEpisodes > 0 && *in.EpisodesWatched > *in.TotalEpisodes {
		errs = append(errs, httpx.FieldError{
			Field: "episodes_watched",
			Error: "episodes_watched cannot exceed total_episodes",
		})
	}
	return errs
}
