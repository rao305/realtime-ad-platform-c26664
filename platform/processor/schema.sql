-- Durable truth for campaigns, spend, idempotency, and local analytics.
-- The ad server never touches these tables on the hot path.

CREATE TABLE IF NOT EXISTS campaigns (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    creative_id     TEXT NOT NULL,
    bid_cpm         NUMERIC(12, 4) NOT NULL CHECK (bid_cpm >= 0),
    daily_budget    NUMERIC(14, 4) NOT NULL CHECK (daily_budget > 0),
    spent_today     NUMERIC(14, 4) NOT NULL DEFAULT 0 CHECK (spent_today >= 0),
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    interests       TEXT[] NOT NULL DEFAULT '{}',
    countries       TEXT[] NOT NULL DEFAULT '{}',
    freq_cap_hour   INT NOT NULL DEFAULT 5 CHECK (freq_cap_hour > 0),
    pacing_mul      NUMERIC(6, 4) NOT NULL DEFAULT 1.0 CHECK (pacing_mul >= 0 AND pacing_mul <= 1),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaigns_active ON campaigns (active) WHERE active;

CREATE TABLE IF NOT EXISTS creatives (
    id           TEXT PRIMARY KEY,
    campaign_id  TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    headline     TEXT NOT NULL,
    body         TEXT NOT NULL DEFAULT '',
    click_url    TEXT NOT NULL DEFAULT ''
);

-- Claim table for at-least-once Kafka delivery. event_id is the natural key.
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     TEXT PRIMARY KEY,
    request_id   TEXT NOT NULL,
    event_type   TEXT NOT NULL,
    campaign_id  TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_request
    ON processed_events (request_id, event_type);

-- Append-only spend ledger for reconciliation / audits.
CREATE TABLE IF NOT EXISTS spend_ledger (
    id           BIGSERIAL PRIMARY KEY,
    event_id     TEXT NOT NULL UNIQUE REFERENCES processed_events(event_id),
    campaign_id  TEXT NOT NULL REFERENCES campaigns(id),
    amount       NUMERIC(14, 6) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Local analytics sink (zero-credential BigQuery stand-in).
CREATE TABLE IF NOT EXISTS analytics_events (
    event_id      TEXT PRIMARY KEY,
    schema_version INT NOT NULL,
    type          TEXT NOT NULL,
    request_id    TEXT NOT NULL,
    campaign_id   TEXT NOT NULL,
    creative_id   TEXT NOT NULL DEFAULT '',
    user_id       TEXT NOT NULL,
    bid_cpm       NUMERIC(12, 4) NOT NULL DEFAULT 0,
    at            TIMESTAMPTZ NOT NULL,
    ingested_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analytics_events_day_campaign
    ON analytics_events (at, campaign_id);

CREATE INDEX IF NOT EXISTS idx_analytics_events_type
    ON analytics_events (type);
