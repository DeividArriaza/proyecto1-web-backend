package models

import "time"

// StatusValues define el dominio del campo `status` tal como lo restringe el
// CHECK constraint del schema SQL.
var StatusValues = []string{"watching", "completed", "dropped", "pending"}

// Series representa una fila de la tabla `series` tal como se serializa en
// JSON hacia el cliente. Los campos opcionales usan punteros para distinguir
// "ausente" de "valor cero".
type Series struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	Genre           *string   `json:"genre"`
	Status          string    `json:"status"`
	EpisodesWatched int       `json:"episodes_watched"`
	TotalEpisodes   *int      `json:"total_episodes"`
	ImagePath       *string   `json:"image_path"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SeriesInput es el payload que aceptan POST y PUT. Mantenemos los mismos
// campos opcionales como punteros para validar "no enviado" vs "enviado vacío".
type SeriesInput struct {
	Title           *string `json:"title"`
	Genre           *string `json:"genre"`
	Status          *string `json:"status"`
	EpisodesWatched *int    `json:"episodes_watched"`
	TotalEpisodes   *int    `json:"total_episodes"`
}

// IsValidStatus indica si `s` es uno de los valores aceptados.
func IsValidStatus(s string) bool {
	for _, v := range StatusValues {
		if v == s {
			return true
		}
	}
	return false
}
