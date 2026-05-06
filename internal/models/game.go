package models

import "time"

// StatusValues define el dominio del campo `status` tal como lo restringe el
// CHECK constraint del schema SQL.
var StatusValues = []string{"playing", "beaten", "dropped", "backlog"}

// Game representa una fila de la tabla `games` tal como se serializa en
// JSON hacia el cliente. Los campos opcionales usan punteros para distinguir
// "ausente" de "valor cero".
type Game struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Genre       *string   `json:"genre"`
	Status      string    `json:"status"`
	HoursPlayed int       `json:"hours_played"`
	TotalHours  *int      `json:"total_hours"`
	ImagePath   *string   `json:"image_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GameInput es el payload que aceptan POST y PUT. Mantenemos los mismos
// campos opcionales como punteros para validar "no enviado" vs "enviado vacío".
type GameInput struct {
	Title       *string `json:"title"`
	Genre       *string `json:"genre"`
	Status      *string `json:"status"`
	HoursPlayed *int    `json:"hours_played"`
	TotalHours  *int    `json:"total_hours"`
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
