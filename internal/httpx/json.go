package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// FieldError describe un único campo inválido en una respuesta de error.
type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// errorOne tiene la forma `{ "error": "...", "field": "..." }` que usa la API
// para errores de un solo campo.
type errorOne struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

// errorMany tiene la forma `{ "errors": [...] }` para validar varios campos
// en una sola respuesta.
type errorMany struct {
	Errors []FieldError `json:"errors"`
}

// JSON envía `body` codificado como JSON con el código indicado.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("json encode: %v", err)
	}
}

// Error responde con un solo error y, opcionalmente, el campo asociado.
func Error(w http.ResponseWriter, status int, message string, field string) {
	JSON(w, status, errorOne{Error: message, Field: field})
}

// Errors responde con múltiples errores de validación en un único 400.
func Errors(w http.ResponseWriter, status int, fes []FieldError) {
	JSON(w, status, errorMany{Errors: fes})
}

// NoContent escribe un 204 sin cuerpo, usado por DELETE.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
