# Games Tracker — Backend

API REST en Go que persiste videojuegos y ratings en PostgreSQL. Sirve también la documentación interactiva (Swagger UI) y los archivos de imagen subidos por los usuarios.

El cliente vive en otro repo: [`proyecto1-web-frontend`](../proyecto1-web-frontend) (debe clonarse al lado de este repo, ver más abajo).

---

## Stack

- **Lenguaje:** Go 1.22 (router `chi`, driver `pgx/v5`)
- **Base de datos:** PostgreSQL 16
- **Infra:** Docker + Docker Compose

---

## Requisitos previos

- Docker (≥ 24.x) con el plugin `compose` v2.
- Git.
- Puertos `8787`, `15432` y `4567` libres en el host (configurables en `.env`).

> No necesitas tener Go ni Postgres instalados localmente; todo corre en contenedores.

---

## Cómo levantar el proyecto

El `docker-compose.yml` vive en este repo y orquesta los **tres** servicios. El servicio `frontend` apunta al repo hermano del cliente, así que ambos deben estar clonados en la misma carpeta:

```text
mi-carpeta/
├── proyecto1-web-backend/      ← este repo
└── proyecto1-web-frontend/     ← repo del cliente
```

### 1. Clonar ambos repos al mismo nivel

```bash
git clone https://github.com/DeividArriaza/proyecto1-web-backend.git
git clone https://github.com/DeividArriaza/proyecto1-web-frontend.git
cd proyecto1-web-backend
```

> Si renombraste la carpeta del frontend, definí `FRONTEND_PATH` en el `.env` con la ruta correcta.

### 2. Configurar variables de entorno

Copia el archivo de ejemplo y ajusta lo que necesites (los valores por defecto sirven para correr en local):

```bash
cp .env.example .env
```

### 3. Construir y levantar

```bash
docker compose up --build
```

La primera vez tardará unos minutos en bajar las imágenes y compilar el binario. En las siguientes solo se reusa la caché.

### 4. URLs resultantes

| Servicio | URL |
|---|---|
| API REST | http://localhost:8787 |
| Frontend (Nginx) | http://localhost:4567 |
| Swagger UI | http://localhost:8787/swagger |
| Spec OpenAPI cruda | http://localhost:8787/openapi.yaml |

Para verificar que todo arrancó bien:

```bash
curl http://localhost:8787/healthz
# → {"status":"ok"}
```

### 5. Bajar el stack

```bash
docker compose down           # detiene y elimina contenedores
docker compose down -v        # …además borra la base de datos y los uploads
```

---

## Estructura del repo

```text
.
├── Dockerfile              # Imagen multi-stage del backend
├── docker-compose.yml      # Orquesta db + api + frontend
├── db/init.sql             # Esquema inicial (games, ratings)
├── go.mod / go.sum         # Dependencias de Go
├── main.go                 # Entry point del servidor HTTP
├── internal/
│   ├── db/                 # Helpers de conexión a Postgres (pgxpool)
│   └── middleware/         # CORS y otros middlewares
├── openapi.yaml            # Spec OpenAPI 3.0 (próxima fase)
└── .env.example            # Plantilla de variables de entorno
```

---

## Endpoints principales

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/healthz` | Healthcheck simple |
| `GET` | `/games` | Listar juegos con `?q=`, `?sort=`, `?order=`, `?page=`, `?limit=` |
| `GET` | `/games/{id}` | Obtener un juego |
| `POST` | `/games` | Crear juego |
| `PUT` | `/games/{id}` | Editar juego |
| `DELETE` | `/games/{id}` | Eliminar juego |
| `POST` | `/games/{id}/image` | Subir portada (`multipart/form-data`, ≤ 1 MB) |
| `GET` | `/games/{id}/rating` | Obtener promedio + conteo |
| `POST` | `/games/{id}/rating` | Registrar voto (1 a 10) |
| `GET` | `/swagger/` | UI interactiva |
| `GET` | `/openapi.yaml` | Spec en YAML |

> La spec completa con request bodies, parámetros y respuestas vive en `openapi.yaml` y se sirve por Swagger UI.

---

## CORS

Todas las rutas (incluyendo el preflight `OPTIONS`) responden con:

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

---

## Solución de problemas

- **`port is already allocated`**: cambia `API_HOST_PORT`, `DB_HOST_PORT` o `FRONTEND_HOST_PORT` en `.env`.
- **El backend no encuentra la DB**: el contenedor de la API espera el healthcheck de Postgres, pero si ves errores de conexión revisa que `DATABASE_URL` use `db` como host (no `localhost`).
- **Cambios en `db/init.sql` no se aplican**: ese script solo corre la primera vez con un volumen vacío. Para forzar la recreación: `docker compose down -v` y vuelve a levantar.

---

## Repositorios relacionados

- Frontend: https://github.com/DeividArriaza/proyecto1-web-frontend/blob/main/README.md
