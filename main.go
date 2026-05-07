package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	appdb "github.com/dsalas/games-tracker-backend/internal/db"
	apph "github.com/dsalas/games-tracker-backend/internal/handlers"
	appmw "github.com/dsalas/games-tracker-backend/internal/middleware"
	appstore "github.com/dsalas/games-tracker-backend/internal/store"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8787"
	}
	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "/app/uploads"
	}
	seedCoversDir := os.Getenv("SEED_COVERS_DIR")
	if seedCoversDir == "" {
		seedCoversDir = "/app/seed-covers"
	}

	// Si el volumen de uploads está vacío, copiamos las portadas de demo que
	// el Dockerfile dejó en /app/seed-covers/. Esto se alinea con el seed SQL
	// (db/seed.sql) que inserta los juegos con image_path apuntando a esos
	// archivos. En reinicios subsiguientes el directorio ya tiene contenido y
	// la función es no-op.
	if err := seedUploadsIfEmpty(uploadsDir, seedCoversDir); err != nil {
		log.Printf("seed uploads: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := appdb.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	log.Println("connected to postgres")

	gameStore := appstore.NewGameStore(pool)
	gameHandler := &apph.GameHandler{Store: gameStore}

	ratingStore := appstore.NewRatingStore(pool)
	ratingHandler := &apph.RatingHandler{Store: ratingStore}

	imageHandler := &apph.ImageHandler{Store: gameStore, UploadsDir: uploadsDir}

	r := chi.NewRouter()
	r.Use(appmw.CORS)

	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	gameHandler.Routes(r)
	ratingHandler.Routes(r)
	imageHandler.Routes(r)

	// Servimos las imágenes subidas como archivos estáticos. La carpeta vive
	// en un volumen Docker para que sobreviva a recreaciones del contenedor.
	uploadsFS := http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir)))
	r.Handle("/uploads/*", uploadsFS)

	docsRoutes(r)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// seedUploadsIfEmpty copia los archivos de `seedDir` a `uploadsDir` solo si
// `uploadsDir` no contiene nada. Pensado para que en el primer arranque del
// contenedor, las portadas que `db/seed.sql` referencia queden disponibles
// sin pasos manuales. Si `seedDir` no existe, la función es un no-op.
func seedUploadsIfEmpty(uploadsDir, seedDir string) error {
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		return err
	}
	existing, err := os.ReadDir(uploadsDir)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	seedEntries, err := os.ReadDir(seedDir)
	if err != nil {
		// Si el directorio de seed no existe (deploy sin demo data) lo dejamos pasar.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	copied := 0
	for _, e := range seedEntries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(seedDir, e.Name())
		dst := filepath.Join(uploadsDir, e.Name())
		if err := copyFile(src, dst); err != nil {
			return err
		}
		copied++
	}
	if copied > 0 {
		log.Printf("seeded %d cover(s) into %s", copied, uploadsDir)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
