# --- Etapa de build ---
FROM golang:1.22-alpine AS build

WORKDIR /src

# Cacheamos las dependencias antes que el resto del código
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el código fuente
COPY . .

# Binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api .

# --- Etapa de runtime ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 app

WORKDIR /app
COPY --from=build /out/api /app/api

# Carpeta de uploads (se monta como volumen desde docker-compose para persistir)
RUN mkdir -p /app/uploads && chown -R app:app /app

USER app

EXPOSE 8787

ENTRYPOINT ["/app/api"]
