-- Geochat schema (family chat + location + SOS). Owned by geochat-svc.
CREATE SCHEMA IF NOT EXISTS geochat;

CREATE TABLE IF NOT EXISTS geochat.messages (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    kind       TEXT NOT NULL DEFAULT 'text' CHECK (kind IN ('text', 'sos')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_geochat_messages_created ON geochat.messages(created_at DESC);

-- Latest known location per user (overwritten on every update -- this
-- app tracks "where is everyone right now", not location history).
CREATE TABLE IF NOT EXISTS geochat.locations (
    user_id    BIGINT PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    latitude   DOUBLE PRECISION NOT NULL,
    longitude  DOUBLE PRECISION NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- SOS triggers are logged separately from geochat.messages (belt and
-- suspenders per the doc: "independent of the main app DB" in spirit --
-- this table plus the append-only file log in cmd/geochat-svc's data
-- dir both record every trigger so it can be audited).
CREATE TABLE IF NOT EXISTS geochat.sos_events (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    latitude     DOUBLE PRECISION,
    longitude    DOUBLE PRECISION,
    message      TEXT,
    telegram_ok  BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
