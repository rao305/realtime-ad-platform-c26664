-- Campaign performance for the dashboard. BigQuery is columnar, so scanning
-- billions of event rows to aggregate a few columns is cheap — the opposite of
-- what you'd want from the OLTP serving database.
SELECT
  campaign_id,
  COUNTIF(type = 'impression')                              AS impressions,
  COUNTIF(type = 'click')                                   AS clicks,
  SAFE_DIVIDE(COUNTIF(type = 'click'),
              COUNTIF(type = 'impression'))                 AS ctr,
  COUNT(DISTINCT user_id)                                   AS reach
FROM `adplatform.analytics.events`
WHERE DATE(at) = CURRENT_DATE()
GROUP BY campaign_id
ORDER BY impressions DESC;
