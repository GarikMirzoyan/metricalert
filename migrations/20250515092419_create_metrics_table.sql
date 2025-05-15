-- +goose Up
CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    value DOUBLE PRECISION
);

-- +goose Down
DROP TABLE IF EXISTS metrics;