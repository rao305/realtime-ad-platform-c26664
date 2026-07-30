-- Local PostgreSQL equivalent of the BigQuery campaign report.
SELECT
  campaign_id,
  COUNT(*) FILTER (WHERE type = 'impression') AS impressions,
  COUNT(*) FILTER (WHERE type = 'click') AS clicks,
  CASE
    WHEN COUNT(*) FILTER (WHERE type = 'impression') = 0 THEN NULL
    ELSE COUNT(*) FILTER (WHERE type = 'click')::numeric
         / COUNT(*) FILTER (WHERE type = 'impression')
  END AS ctr,
  COUNT(DISTINCT user_id) AS reach,
  SUM(CASE WHEN type = 'impression' THEN bid_cpm / 1000.0 ELSE 0 END) AS spend
FROM analytics_events
WHERE at::date = CURRENT_DATE
GROUP BY campaign_id
ORDER BY impressions DESC;
