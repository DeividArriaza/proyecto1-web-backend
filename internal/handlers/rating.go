package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dsalas/games-tracker-backend/internal/httpx"
	"github.com/dsalas/games-tracker-backend/internal/store"
)

// RatingHandler agrupa los endpoints de rating sobre /games/{id}/rating.
// Comparte el RatingStore para todas las consultas y verificaciones de
// existencia del juego asociado.
type RatingHandler struct {
	Store *store.RatingStore
}

// Routes monta los endpoints bajo el router que recibe.
func (h *RatingHandler) Routes(r chi.Router) {
	r.Get("/games/{id}/rating", h.Get)
	r.Post("/games/{id}/rating", h.Submit)
}

// ratingInput es el payload aceptado por POST /games/{id}/rating.
type ratingInput struct {
	Score *int `json:"score"`
}

func (h *RatingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	exists, err := h.Store.GameExists(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to check game", "")
		return
	}
	if !exists {
		httpx.Error(w, http.StatusNotFound, "game not found", "")
		return
	}

	sum, err := h.Store.Summary(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load rating", "")
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

func (h *RatingHandler) Submit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var in ratingInput
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON body: "+err.Error(), "")
		return
	}
	if in.Score == nil {
		httpx.Error(w, http.StatusBadRequest, "score is required", "score")
		return
	}
	if *in.Score < 1 || *in.Score > 10 {
		httpx.Error(w, http.StatusBadRequest, "score must be between 1 and 10", "score")
		return
	}

	exists, err := h.Store.GameExists(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to check game", "")
		return
	}
	if !exists {
		httpx.Error(w, http.StatusNotFound, "game not found", "")
		return
	}

	if err := h.Store.Insert(r.Context(), id, *in.Score); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to record rating", "")
		return
	}

	sum, err := h.Store.Summary(r.Context(), id)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load rating", "")
		return
	}
	httpx.JSON(w, http.StatusCreated, sum)
}
