"""Real-time stream processor.

Consumes the `ad-events` topic, aggregates spend + engagement per campaign in a
short window, and writes two things back:
  • running spend → PostgreSQL (durable truth for billing)
  • a pacing multiplier → Redis (so the Go ad server throttles live)

This is the write path's brain: it turns a firehose of events into the feedback
that paces campaigns and the rollups that feed analytics.
"""
from kafka import KafkaConsumer
import json
import time

from store import record_spend, set_pacing, already_seen

CPM = 1000  # cost is per-mille (per 1000 impressions)


def run() -> None:
    consumer = KafkaConsumer(
        "ad-events",
        bootstrap_servers="kafka:9092",
        group_id="spend-aggregator",      # consumer group = scalable, resumable
        enable_auto_commit=False,         # commit only after we've processed
        auto_offset_reset="earliest",
    )

    window: dict[str, int] = {}           # campaign_id -> impressions this window
    window_start = time.time()

    for msg in consumer:
        ev = json.loads(msg.value)

        # Idempotency: at-least-once delivery means we may see an event twice.
        if already_seen(ev["request_id"], ev["type"]):
            consumer.commit()
            continue

        if ev["type"] == "impression":
            window[ev["campaign_id"]] = window.get(ev["campaign_id"], 0) + 1

        # Flush the window every second: persist spend + recompute pacing.
        if time.time() - window_start >= 1.0:
            for campaign_id, impressions in window.items():
                spend = impressions / CPM  # simplified: bid assumed normalized
                total = record_spend(campaign_id, spend)        # -> PostgreSQL
                set_pacing(campaign_id, total)                  # -> Redis
            window.clear()
            window_start = time.time()

        consumer.commit()  # advance offset only after successful processing
