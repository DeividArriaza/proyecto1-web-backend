package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RatingStore expone las operaciones de la tabla `ratings`. Reutiliza el
// mismo pgxpool del resto de la aplicación.
type RatingStore struct {
	pool *pgxpool.Pool
}

func NewRatingStore(pool *pgxpool.Pool) *RatingStore {
	return &RatingStore{pool: pool}
}

// RatingSummary es lo que devuelven tanto GET como POST /games/:id/rating.
type RatingSummary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// GameExists indica si hay un juego con ese id, para que los handlers de
// rating puedan responder 404 antes de intentar el INSERT (evita que el
// error de FK se filtre como 500).
func (s *RatingStore) GameExists(ctx context.Context, gameID int) (bool, error) {
	var one int
	err := s.pool.QueryRow(ctx, "SELECT 1 FROM games WHERE id = $1", gameID).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check game exists: %w", err)
	}
	return true, nil
}

// Insert registra un voto anónimo. El handler debe validar el rango 1..10
// antes de llamar; igualmente el CHECK de la DB lo bloquearía.
func (s *RatingStore) Insert(ctx context.Context, gameID int, score int) error {
	_, err := s.pool.Exec(ctx,
		"INSERT INTO ratings (game_id, score) VALUES ($1, $2)",
		gameID, score,
	)
	if err != nil {
		return fmt.Errorf("insert rating: %w", err)
	}
	return nil
}

// Summary calcula el promedio y conteo para un juego. Devuelve ceros si
// todavía no tiene votos (lo que es válido aun cuando el juego sí existe).
func (s *RatingStore) Summary(ctx context.Context, gameID int) (RatingSummary, error) {
	var sum RatingSummary
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(AVG(score)::float, 0) AS average,
		       COUNT(*)                       AS count
		  FROM ratings
		 WHERE game_id = $1`, gameID,
	).Scan(&sum.Average, &sum.Count)
	if err != nil {
		return RatingSummary{}, fmt.Errorf("rating summary: %w", err)
	}
	return sum, nil
}
