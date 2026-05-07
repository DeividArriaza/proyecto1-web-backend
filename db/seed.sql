-- Datos de demo para el deploy público.
-- Postgres ejecuta los archivos de /docker-entrypoint-initdb.d/ una sola vez
-- (cuando el volumen está vacío), así que esta semilla solo se carga en la
-- primera puesta en marcha. Si querés volver a partir de cero: `docker
-- compose down -v && docker compose up --build`.
--
-- Las rutas de `image_path` apuntan a nombres deterministas en /uploads/.
-- El backend, al arrancar, copia los archivos de /app/seed-covers/ al
-- volumen `api_uploads` solo si está vacío, así esos paths quedan servibles
-- por /uploads/* sin pasos manuales.

INSERT INTO games (title, genre, status, hours_played, total_hours, image_path) VALUES
  ('League of Legends',           'MOBA',                    'playing', 1200, NULL, '/uploads/lol.png'),
  ('Valorant',                    'Tactical FPS',            'playing',  340, NULL, '/uploads/valorant.png'),
  ('Clair Obscur: Expedition 33', 'JRPG por turnos',         'beaten',    48,   35, '/uploads/expedition_33.jpg'),
  ('Elden Ring Nightreign',       'Action RPG / Roguelite',  'playing',   22,   40, '/uploads/elden_ring_nightreign.jpg'),
  ('Lethal Company',              'Co-op horror',            'playing',   15, NULL, NULL),
  ('PEAK',                        'Co-op climbing',          'backlog',    0, NULL, NULL),
  ('Sekiro: Shadows Die Twice',   'Soulslike',               'beaten',    58,   40, NULL),
  ('Hades II',                    'Roguelite',               'playing',   35, NULL, NULL),
  ('Helldivers 2',                'Co-op shooter',           'dropped',   42, NULL, NULL),
  ('Baldur''s Gate 3',            'CRPG',                    'backlog',    0,  120, NULL);
