"""Real-time spend aggregator.

Consumes `ad-events`, claims each event idempotently in PostgreSQL, accrues
impression spend, and pushes pacing multipliers back to Redis.
"""
from __future__ import annotations

import json
import logging
import signal
import sys
import time
from pathlib import Path

from prometheus_client import Counter, Histogram, start_http_server

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "contracts" / "python"))
sys.path.insert(0, str(Path(__file__).resolve().parent))

from config import kafka_brokers, kafka_dlq_topic, kafka_topic  # noqa: E402
from contracts import EVENT_IMPRESSION, validate_event  # noqa: E402
from store import claim_event, record_impression, set_pacing  # noqa: E402

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("spend-processor")

EVENTS_TOTAL = Counter("processor_events_total", "Events handled", ["result"])
SPEND_SECONDS = Histogram("processor_spend_seconds", "Time to record spend")
DUPLICATES = Counter("processor_duplicates_total", "Duplicate events suppressed")

_running = True


def _stop(*_args) -> None:
    global _running
    _running = False


def impression_cost(ev: dict) -> float:
    bid = float(ev.get("bid_cpm") or 0)
    if bid <= 0:
        bid = 1.0
    return bid / 1000.0


def process_event(ev: dict, producer=None) -> None:
    try:
        validate_event(ev)
    except ValueError as exc:
        EVENTS_TOTAL.labels("invalid").inc()
        log.warning("invalid event: %s payload=%s", exc, ev)
        if producer is not None:
            producer.send(kafka_dlq_topic(), json.dumps({"error": str(exc), "event": ev}).encode())
        return

    event_id = ev["event_id"]
    if ev["type"] == EVENT_IMPRESSION:
        amount = impression_cost(ev)
        with SPEND_SECONDS.time():
            claimed, spent, budget, active = record_impression(
                event_id,
                ev["request_id"],
                ev["campaign_id"],
                amount,
            )
        if not claimed:
            DUPLICATES.inc()
            EVENTS_TOTAL.labels("duplicate").inc()
            return
        set_pacing(ev["campaign_id"], spent, budget, active)
    elif not claim_event(event_id, ev["request_id"], ev["type"], ev["campaign_id"]):
        DUPLICATES.inc()
        EVENTS_TOTAL.labels("duplicate").inc()
        return

    EVENTS_TOTAL.labels("ok").inc()


def run() -> None:
    from kafka import KafkaConsumer, KafkaProducer

    start_http_server(9101)
    signal.signal(signal.SIGINT, _stop)
    signal.signal(signal.SIGTERM, _stop)

    consumer = KafkaConsumer(
        kafka_topic(),
        bootstrap_servers=kafka_brokers(),
        group_id="spend-aggregator",
        enable_auto_commit=False,
        auto_offset_reset="earliest",
        value_deserializer=lambda b: json.loads(b.decode("utf-8")),
        consumer_timeout_ms=1000,
    )
    producer = KafkaProducer(bootstrap_servers=kafka_brokers())

    log.info("spend processor started topic=%s", kafka_topic())
    while _running:
        polled = consumer.poll(timeout_ms=1000)
        if not polled:
            continue
        for _tp, messages in polled.items():
            for msg in messages:
                try:
                    process_event(msg.value, producer)
                    consumer.commit()
                except Exception:
                    log.exception("failed processing offset=%s", msg.offset)
                    time.sleep(0.5)
                    raise
    consumer.close()
    producer.close()
    log.info("spend processor stopped")


if __name__ == "__main__":
    run()
