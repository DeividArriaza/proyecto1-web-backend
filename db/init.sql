-- Esquema inicial de Series Tracker.
-- Postgres ejecuta los archivos de /docker-entrypoint-initdb.d/ una sola vez,
-- la primera vez que arranca con un volumen de datos vacío.

CREATE TABLE IF NOT EXISTS series (
    id               SERIAL PRIMARY KEY,
    title            VARCHAR(255) NOT NULL,
    genre            VARCHAR(100),
    status           VARCHAR(50) NOT NULL
                      CHECK (status IN ('watching', 'completed', 'dropped', 'pending')),
    episodes_watched INT DEFAULT 0,
    total_episodes   INT,
    image_path       VARCHAR(500),
    created_at       TIMESTAMP DEFAULT NOW(),
    updated_at       TIMESTAMP DEFAULT NOW()
);

-- Índice de apoyo para búsquedas por título (ILIKE) y para el orden alfabético.
CREATE INDEX IF NOT EXISTS series_title_idx ON series (LOWER(title));

CREATE TABLE IF NOT EXISTS ratings (
    id         SERIAL PRIMARY KEY,
    series_id  INT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    score      SMALLINT NOT NULL CHECK (score BETWEEN 1 AND 10),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Índice para calcular promedio/conteo por serie de manera eficiente.
CREATE INDEX IF NOT EXISTS ratings_series_id_idx ON ratings (series_id);
