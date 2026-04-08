-- +goose Up
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL,
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    title TEXT NOT NULL,
    description TEXT,
    starts_at TIMESTAMPTZ NOT NULL,
    duration_minutes INT NOT NULL,
    location_name TEXT,
    location_coords TEXT,
    max_participants INT,
    status TEXT NOT NULL DEFAULT 'draft',
    event_code INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_status CHECK (status IN ('draft', 'active', 'cancelled', 'completed'))
);

-- +goose Down
DROP TABLE IF EXISTS events;