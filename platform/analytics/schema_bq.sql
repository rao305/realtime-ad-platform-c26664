-- BigQuery DDL for the optional analytics sink.
CREATE TABLE IF NOT EXISTS `adplatform.analytics.events` (
  event_id STRING NOT NULL,
  schema_version INT64 NOT NULL,
  type STRING NOT NULL,
  request_id STRING NOT NULL,
  campaign_id STRING NOT NULL,
  creative_id STRING,
  user_id STRING NOT NULL,
  bid_cpm FLOAT64,
  at TIMESTAMP NOT NULL
)
PARTITION BY DATE(at)
CLUSTER BY campaign_id, type;
