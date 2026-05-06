package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/dsalas/games-tracker-backend/internal/httpx"
	"github.com/dsalas/games-tracker-backend/internal/store"
)

// MaxImageSize es el tope que exige la spec: 1 MiB por archivo subido.
const MaxImageSize int64 = 1 << 20

// allowedMIMEs lista los tipos aceptados y su extensión canónica. La
// validación se hace mirando el contenido real del archivo (DetectContentType),
// no el header `Content-Type` que envíe el cliente.
var allowedMIMEs = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ImageHandler agrupa los endpoints de subida de portadas. Necesita acceso
// al GameStore para verificar existencia, leer la ruta vigente y actualizarla.
type ImageHandler struct {
	Store      *store.GameStore
	UploadsDir string
}

// Routes monta POST /games/{id}/image bajo el router que recibe.
func (h *ImageHandler) Routes(r chi.Router) {
	r.Post("/games/{id}/image", h.Upload)
}

// Upload guarda la imagen en disco, actualiza games.image_path y borra la
// imagen anterior si la había. Devuelve 400 cuando el archivo no es válido,
// 404 cuando el juego no existe y 500 en errores internos.
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	// Cortamos el body antes de parsear para no comer memoria con cargas grandes.
	// El extra de 4 KB cubre el overhead del propio multipart (boundaries, headers).
	r.Body = http.MaxBytesReader(w, r.Body, MaxImageSize+4096)

	if err := r.ParseMultipartForm(MaxImageSize); err != nil {
		httpx.Error(w, http.StatusBadRequest,
			"image must be at most 1 MB and use multipart/form-data", "image")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "image field is required", "image")
		return
	}
	defer file.Close()

	if header.Size > MaxImageSize {
		httpx.Error(w, http.StatusBadRequest, "image must be at most 1 MB", "image")
		return
	}

	// Sniffeo del MIME real con los primeros 512 bytes — más confiable que
	// confiar en el header del cliente o en la extensión.
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	mime := http.DetectContentType(head[:n])
	ext, allowed := allowedMIMEs[mime]
	if !allowed {
		httpx.Error(w, http.StatusBadRequest,
			"image must be jpeg, png or webp", "image")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to read image", "")
		return
	}

	// Verificamos existencia y conservamos la ruta actual para borrar el
	// archivo viejo después de un escritura exitosa.
	current, err := h.Store.CurrentImagePath(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "game not found", "")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load game", "")
		return
	}

	// Nombre determinístico-y-único: id del juego + nano timestamp + extensión
	// canónica. Evita colisiones entre subidas concurrentes y descarta nombres
	// arbitrarios del usuario (path traversal, caracteres raros).
	name := fmt.Sprintf("%d_%d%s", id, time.Now().UnixNano(), ext)
	targetPath := filepath.Join(h.UploadsDir, name)

	if err := os.MkdirAll(h.UploadsDir, 0o755); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to prepare uploads dir", "")
		return
	}

	out, err := os.Create(targetPath)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to save image", "")
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		_ = os.Remove(targetPath)
		httpx.Error(w, http.StatusInternalServerError, "failed to save image", "")
		return
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(targetPath)
		httpx.Error(w, http.StatusInternalServerError, "failed to save image", "")
		return
	}

	publicPath := "/uploads/" + name
	if err := h.Store.SetImagePath(r.Context(), id, publicPath); err != nil {
		_ = os.Remove(targetPath)
		if errors.Is(err, store.ErrNotFound) {
			// Carrera improbable: el juego se borró entre el check y el update.
			httpx.Error(w, http.StatusNotFound, "game not found", "")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to update image_path", "")
		return
	}

	// Limpieza del archivo anterior. Best-effort: si falla, lo dejamos como
	// huérfano, pero validamos que la ruta a borrar siga siendo dentro del
	// directorio de uploads para evitar borrados fuera del volumen.
	if current != nil && *current != "" {
		oldName := strings.TrimPrefix(*current, "/uploads/")
		oldPath := filepath.Join(h.UploadsDir, oldName)
		rel, relErr := filepath.Rel(h.UploadsDir, oldPath)
		if relErr == nil && !strings.HasPrefix(rel, "..") && oldPath != targetPath {
			_ = os.Remove(oldPath)
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"image_path": publicPath})
}
