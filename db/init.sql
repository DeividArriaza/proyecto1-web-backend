-- Esquema inicial de Games Tracker.
-- Postgres ejecuta los archivos de /docker-entrypoint-initdb.d/ una sola vez,
-- la primera vez que arranca con un volumen de datos vacío.

CREATE TABLE IF NOT EXISTS games (
    id           SERIAL PRIMARY KEY,
    title        VARCHAR(255) NOT NULL,
    genre        VARCHAR(100),
    status       VARCHAR(50) NOT NULL
                  CHECK (status IN ('playing', 'beaten', 'dropped', 'backlog')),
    hours_played INT DEFAULT 0,
    total_hours  INT,
    image_path   VARCHAR(500),
    created_at   TIMESTAMP DEFAULT NOW(),
    updated_at   TIMESTAMP DEFAULT NOW()
);

-- Índice de apoyo para búsquedas por título (ILIKE) y para el orden alfabético.
CREATE INDEX IF NOT EXISTS games_title_idx ON games (LOWER(title));

CREATE TABLE IF NOT EXISTS ratings (
    id         SERIAL PRIMARY KEY,
    game_id    INT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    score      SMALLINT NOT NULL CHECK (score BETWEEN 1 AND 10),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Índice para calcular promedio/conteo por juego de manera eficiente.
CREATE INDEX IF NOT EXISTS ratings_game_id_idx ON ratings (game_id);
