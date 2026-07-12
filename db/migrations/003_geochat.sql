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
--
-- There's no third-party fallback (no Telegram/WhatsApp bot) -- the
-- live websocket broadcast is the only delivery path SOS has, so
-- recipients_online records how many other family members were
-- actually connected and able to see the alert the instant it fired.
CREATE TABLE IF NOT EXISTS geochat.sos_events (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            BIGINT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    latitude           DOUBLE PRECISION,
    longitude          DOUBLE PRECISION,
    message            TEXT,
    recipients_online  INT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
