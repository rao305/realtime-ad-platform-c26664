"""Optional BigQuery analytics sink.

Enable with ANALYTICS_SINK=bigquery and Application Default Credentials.
Batch inserts flush on size or time; offsets commit only after a successful write.
"""
from __future__ import annotations

import json
import logging
import os
import signal
import sys
import time
from pathlib import Path

from kafka import KafkaConsumer
from prometheus_client import Counter, start_http_server

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "contracts" / "python"))
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "processor"))

from config import kafka_brokers, kafka_topic  # noqa: E402
from contracts import validate_event  # noqa: E402

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("bq-sink")

SINK_TOTAL = Counter("bq_sink_events_total", "BigQuery sink events", ["result"])
_running = True


def _stop(*_args) -> None:
    global _running
    _running = False


def flush(client, table: str, batch: list[dict]) -> None:
    if not batch:
        return
    errors = client.insert_rows_json(table, batch)
    if errors:
        raise RuntimeError(f"BigQuery insert errors: {errors}")


def run() -> None:
    from google.cloud import bigquery

    start_http_server(9104)
    signal.signal(signal.SIGINT, _stop)
    signal.signal(signal.SIGTERM, _stop)

    table = os.getenv("BQ_TABLE", "adplatform.analytics.events")
    batch_size = int(os.getenv("BQ_BATCH_SIZE", "500"))
    flush_sec = float(os.getenv("BQ_FLUSH_SEC", "2"))
    client = bigquery.Client()

    consumer = KafkaConsumer(
        kafka_topic(),
        bootstrap_servers=kafka_brokers(),
        group_id="bq-sink",
        enable_auto_commit=False,
        auto_offset_reset="earliest",
        value_deserializer=lambda b: json.loads(b.decode("utf-8")),
        consumer_timeout_ms=1000,
    )

    batch: list[dict] = []
    last_flush = time.time()
    log.info("bq sink started table=%s", table)

    while _running:
        polled = consumer.poll(timeout_ms=500)
        for _tp, messages in polled.items():
            for msg in messages:
                ev = msg.value
                try:
                    validate_event(ev)
                except ValueError:
                    SINK_TOTAL.labels("invalid").inc()
                    consumer.commit()
                    continue
                row = {
                    "event_id": ev["event_id"],
                    "schema_version": ev["schema_version"],
                    "type": ev["type"],
                    "request_id": ev["request_id"],
                    "campaign_id": ev["campaign_id"],
                    "creative_id": ev.get("creative_id", ""),
                    "user_id": ev["user_id"],
                    "bid_cpm": float(ev.get("bid_cpm") or 0),
                    "at": ev["at"],
                }
                batch.append(row)
                if len(batch) >= batch_size:
                    flush(client, table, batch)
                    batch.clear()
                    consumer.commit()
                    last_flush = time.time()
                    SINK_TOTAL.labels("ok").inc()

        if batch and time.time() - last_flush >= flush_sec:
            try:
                flush(client, table, batch)
                batch.clear()
                consumer.commit()
                last_flush = time.time()
                SINK_TOTAL.labels("ok").inc()
            except Exception:
                SINK_TOTAL.labels("error").inc()
                log.exception("bq flush failed")
                time.sleep(1)

    # Final flush on shutdown.
    if batch:
        flush(client, table, batch)
        consumer.commit()
    consumer.close()
    log.info("bq sink stopped")


if __name__ == "__main__":
    run()
