package main

// Sirve la documentación interactiva (Swagger UI) y la spec OpenAPI en crudo.
// Tanto el YAML como los assets de swagger-ui-dist viven embebidos en el
// binario, así la imagen Docker no depende de archivos externos en runtime.

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed openapi.yaml
var openapiSpec []byte

//go:embed swagger-ui
var swaggerUIFS embed.FS

// docsRoutes monta:
//   - GET /openapi.yaml    → la spec en crudo
//   - GET /swagger         → redirect a /swagger/
//   - GET /swagger/        → la UI interactiva
//   - GET /swagger/<file>  → assets estáticos (CSS, JS) embebidos
func docsRoutes(r chi.Router) {
	r.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(openapiSpec)
	})

	// chi usa /swagger sin slash final como ruta literal y /swagger/* como prefijo.
	r.Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/", http.StatusMovedPermanently)
	})

	// Subárbol del FS embebido apuntando a la carpeta swagger-ui/.
	sub, err := fs.Sub(swaggerUIFS, "swagger-ui")
	if err != nil {
		// Solo puede pasar en build-time si el embed se rompe; falla ruidoso.
		panic("docs: cannot open swagger-ui sub-fs: " + err.Error())
	}

	// http.FileServer maneja index.html automáticamente cuando la ruta termina en /.
	fileServer := http.StripPrefix("/swagger/", http.FileServer(http.FS(sub)))
	r.Handle("/swagger/*", fileServer)
}
