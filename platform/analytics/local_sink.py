"""Local PostgreSQL analytics sink — zero-credential BigQuery stand-in.

A separate consumer group on `ad-events` lands raw events into analytics_events
so dashboards work without GCP credentials.
"""
from __future__ import annotations

import json
import logging
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
from store import insert_analytics_event  # noqa: E402

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("local-analytics")

SINK_TOTAL = Counter("analytics_sink_events_total", "Analytics sink events", ["result"])
_running = True


def _stop(*_args) -> None:
    global _running
    _running = False


def run() -> None:
    start_http_server(9103)
    signal.signal(signal.SIGINT, _stop)
    signal.signal(signal.SIGTERM, _stop)

    consumer = KafkaConsumer(
        kafka_topic(),
        bootstrap_servers=kafka_brokers(),
        group_id="pg-analytics-sink",
        enable_auto_commit=False,
        auto_offset_reset="earliest",
        value_deserializer=lambda b: json.loads(b.decode("utf-8")),
        consumer_timeout_ms=1000,
    )
    log.info("local analytics sink started")
    while _running:
        polled = consumer.poll(timeout_ms=1000)
        for _tp, messages in polled.items():
            for msg in messages:
                ev = msg.value
                try:
                    validate_event(ev)
                    insert_analytics_event(ev)
                    consumer.commit()
                    SINK_TOTAL.labels("ok").inc()
                except Exception:
                    SINK_TOTAL.labels("error").inc()
                    log.exception("analytics insert failed")
                    time.sleep(0.5)
                    raise
    consumer.close()
    log.info("local analytics sink stopped")


if __name__ == "__main__":
    run()
