-- Campaign performance for the dashboard (BigQuery).
SELECT
  campaign_id,
  COUNTIF(type = 'impression') AS impressions,
  COUNTIF(type = 'click') AS clicks,
  SAFE_DIVIDE(COUNTIF(type = 'click'), COUNTIF(type = 'impression')) AS ctr,
  COUNT(DISTINCT user_id) AS reach,
  SUM(IF(type = 'impression', bid_cpm / 1000, 0)) AS spend
FROM `adplatform.analytics.events`
WHERE DATE(at) = CURRENT_DATE()
GROUP BY campaign_id
ORDER BY impressions DESC;
