-- The durable truth. The ad server never touches this on the hot path; the
-- stream processor writes spend here, and the cache refresher reads campaigns
-- from here into Redis.

CREATE TABLE campaigns (
    id            TEXT PRIMARY KEY,
    creative_id   TEXT NOT NULL,
    bid_cpm       NUMERIC NOT NULL,
    daily_budget  NUMERIC NOT NULL,
    spent_today   NUMERIC NOT NULL DEFAULT 0,
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    interests     TEXT[] NOT NULL DEFAULT '{}',
    countries     TEXT[] NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_campaigns_active ON campaigns (active) WHERE active;
