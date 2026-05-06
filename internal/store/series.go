package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dsalas/series-tracker-backend/internal/models"
)

// ErrNotFound se devuelve cuando una operación apuntaba a un id que no existe.
var ErrNotFound = errors.New("series not found")

// SeriesStore expone las operaciones de persistencia para series sobre un
// pgxpool. Mantenemos el SQL aquí para que los handlers no toquen la DB
// directamente.
type SeriesStore struct {
	pool *pgxpool.Pool
}

func NewSeriesStore(pool *pgxpool.Pool) *SeriesStore {
	return &SeriesStore{pool: pool}
}

// ListOptions agrupa los filtros y la paginación que admite GET /series.
// Los valores se asumen ya validados/normalizados por el handler.
type ListOptions struct {
	Q     string // búsqueda parcial sobre el título (ILIKE)
	Sort  string // columna de orden: title | created_at | status
	Order string // asc | desc
	Page  int    // 1-indexed
	Limit int    // 1..100
}

// allowedSortColumns evita SQL injection en el ORDER BY: solo aceptamos
// columnas conocidas.
var allowedSortColumns = map[string]string{
	"title":      "title",
	"created_at": "created_at",
	"status":     "status",
}

func (o ListOptions) orderClause() string {
	col, ok := allowedSortColumns[o.Sort]
	if !ok {
		col = "created_at"
	}
	dir := strings.ToUpper(o.Order)
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	// Empate determinista por id para que la paginación sea estable.
	return fmt.Sprintf("ORDER BY %s %s, id ASC", col, dir)
}

// List retorna las series que cumplen `opts` junto con el total que coincide
// con el filtro `q` (sin aplicar paginación).
func (s *SeriesStore) List(ctx context.Context, opts ListOptions) ([]models.Series, int, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit < 1 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	args := []any{}
	where := ""
	if q := strings.TrimSpace(opts.Q); q != "" {
		args = append(args, "%"+q+"%")
		where = "WHERE title ILIKE $1"
	}

	var total int
	countSQL := "SELECT COUNT(*) FROM series " + where
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count series: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, opts.Limit, (opts.Page-1)*opts.Limit)

	listSQL := fmt.Sprintf(`
		SELECT id, title, genre, status, episodes_watched, total_episodes,
		       image_path, created_at, updated_at
		  FROM series
		%s
		%s
		LIMIT $%d OFFSET $%d`,
		where, opts.orderClause(), limitIdx, offsetIdx,
	)

	rows, err := s.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query series: %w", err)
	}
	defer rows.Close()

	out := make([]models.Series, 0)
	for rows.Next() {
		var sx models.Series
		if err := rows.Scan(
			&sx.ID, &sx.Title, &sx.Genre, &sx.Status, &sx.EpisodesWatched,
			&sx.TotalEpisodes, &sx.ImagePath, &sx.CreatedAt, &sx.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan series: %w", err)
		}
		out = append(out, sx)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iter series: %w", err)
	}
	return out, total, nil
}

func (s *SeriesStore) GetByID(ctx context.Context, id int) (*models.Series, error) {
	const q = `
		SELECT id, title, genre, status, episodes_watched, total_episodes,
		       image_path, created_at, updated_at
		  FROM series
		 WHERE id = $1`

	var sx models.Series
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&sx.ID, &sx.Title, &sx.Genre, &sx.Status, &sx.EpisodesWatched,
		&sx.TotalEpisodes, &sx.ImagePath, &sx.CreatedAt, &sx.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get series: %w", err)
	}
	return &sx, nil
}

// Insert crea una serie y devuelve el registro completo (con timestamps e id
// asignados por la DB).
func (s *SeriesStore) Insert(ctx context.Context, in models.SeriesInput) (*models.Series, error) {
	const q = `
		INSERT INTO series (title, genre, status, episodes_watched, total_episodes)
		     VALUES ($1, $2, $3, COALESCE($4, 0), $5)
		  RETURNING id, title, genre, status, episodes_watched, total_episodes,
		            image_path, created_at, updated_at`

	var sx models.Series
	err := s.pool.QueryRow(ctx, q,
		deref(in.Title),
		in.Genre,
		deref(in.Status),
		in.EpisodesWatched,
		in.TotalEpisodes,
	).Scan(
		&sx.ID, &sx.Title, &sx.Genre, &sx.Status, &sx.EpisodesWatched,
		&sx.TotalEpisodes, &sx.ImagePath, &sx.CreatedAt, &sx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert series: %w", err)
	}
	return &sx, nil
}

// Update sobreescribe los campos editables. Mantenemos image_path intacto
// porque se gestiona por el endpoint de subida de imagen.
func (s *SeriesStore) Update(ctx context.Context, id int, in models.SeriesInput) (*models.Series, error) {
	const q = `
		UPDATE series
		   SET title            = $1,
		       genre            = $2,
		       status           = $3,
		       episodes_watched = COALESCE($4, episodes_watched),
		       total_episodes   = $5,
		       updated_at       = NOW()
		 WHERE id = $6
		 RETURNING id, title, genre, status, episodes_watched, total_episodes,
		           image_path, created_at, updated_at`

	var sx models.Series
	err := s.pool.QueryRow(ctx, q,
		deref(in.Title),
		in.Genre,
		deref(in.Status),
		in.EpisodesWatched,
		in.TotalEpisodes,
		id,
	).Scan(
		&sx.ID, &sx.Title, &sx.Genre, &sx.Status, &sx.EpisodesWatched,
		&sx.TotalEpisodes, &sx.ImagePath, &sx.CreatedAt, &sx.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update series: %w", err)
	}
	return &sx, nil
}

// Delete devuelve ErrNotFound si no había una fila con ese id.
func (s *SeriesStore) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM series WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete series: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetImagePath actualiza solo la ruta de la imagen (lo usa el endpoint de
// subida para no pisar el resto del registro).
func (s *SeriesStore) SetImagePath(ctx context.Context, id int, path string) error {
	tag, err := s.pool.Exec(ctx,
		"UPDATE series SET image_path = $1, updated_at = NOW() WHERE id = $2",
		path, id,
	)
	if err != nil {
		return fmt.Errorf("set image_path: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CurrentImagePath devuelve la ruta vigente (o nil) para que el handler
// de subida pueda borrar el archivo anterior cuando llega uno nuevo.
func (s *SeriesStore) CurrentImagePath(ctx context.Context, id int) (*string, error) {
	var path *string
	err := s.pool.QueryRow(ctx, "SELECT image_path FROM series WHERE id = $1", id).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get current image_path: %w", err)
	}
	return path, nil
}

// deref devuelve el valor del puntero o el cero del tipo si es nil. Usado
// para mapear `*string` opcionales al SQL sin sufrir nil panics.
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
