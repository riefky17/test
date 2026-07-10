-- Finance Tracker schema (IDR). Owned by finance-svc only.
CREATE SCHEMA IF NOT EXISTS finance;

CREATE TABLE IF NOT EXISTS finance.categories (
    id       BIGSERIAL PRIMARY KEY,
    name     TEXT NOT NULL UNIQUE,
    kind     TEXT NOT NULL CHECK (kind IN ('income', 'expense')),
    icon     TEXT NOT NULL DEFAULT '💰'
);

CREATE TABLE IF NOT EXISTS finance.transactions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    category_id   BIGINT NOT NULL REFERENCES finance.categories(id),
    -- Stored in whole Rupiah (IDR has no minor unit in everyday use).
    amount_idr    BIGINT NOT NULL CHECK (amount_idr <> 0),
    note          TEXT,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_finance_tx_user_occurred ON finance.transactions(user_id, occurred_at DESC);

INSERT INTO finance.categories (name, kind, icon) VALUES
    ('Salary',        'income',  '💵'),
    ('Groceries',     'expense', '🛒'),
    ('Utilities',     'expense', '💡'),
    ('Transport',     'expense', '🚗'),
    ('Health',        'expense', '💊'),
    ('Education',     'expense', '📚'),
    ('Entertainment', 'expense', '🎬'),
    ('Other',         'expense', '📦')
ON CONFLICT (name) DO NOTHING;
