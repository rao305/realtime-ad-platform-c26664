"""Postgres → Redis cache synchronizer.

Rebuilds the serving view so the ad server never reads PostgreSQL on the hot
path. Safe to restart: every cycle rewrites campaign hashes and indexes.
"""
from __future__ import annotations

import logging
import os
import signal
import sys
import time
from pathlib import Path

import psycopg
import redis
from prometheus_client import Counter, Gauge, start_http_server

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "processor"))
from config import postgres_dsn, redis_url  # noqa: E402

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("cache-sync")

SYNC_TOTAL = Counter("cache_sync_total", "Cache sync cycles", ["result"])
CAMPAIGNS_GAUGE = Gauge("cache_sync_campaigns", "Active campaigns published to Redis")

_running = True


def _stop(*_args) -> None:
    global _running
    _running = False


def load_campaigns(conn):
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT id, creative_id, bid_cpm, active, interests, countries,
                   freq_cap_hour, pacing_mul, spent_today, daily_budget
            FROM campaigns
            """
        )
        cols = [d.name for d in cur.description]
        return [dict(zip(cols, row)) for row in cur.fetchall()]


def _clear_prefix(r: redis.Redis, prefix: str) -> None:
    cursor = 0
    while True:
        cursor, keys = r.scan(cursor=cursor, match=f"{prefix}*", count=200)
        if keys:
            r.delete(*keys)
        if cursor == 0:
            break


def publish(r: redis.Redis, campaigns: list[dict]) -> int:
    # Rebuild indexes from scratch so deleted / paused campaigns disappear.
    _clear_prefix(r, "idx:interest:")
    _clear_prefix(r, "idx:country:")

    pipe = r.pipeline()
    active_count = 0
    for c in campaigns:
        cid = c["id"]
        active = (
            bool(c["active"])
            and float(c["pacing_mul"]) > 0
            and float(c["spent_today"]) < float(c["daily_budget"])
        )
        if active:
            active_count += 1
        mapping = {
            "creative_id": c["creative_id"],
            "bid_cpm": f"{float(c['bid_cpm']):.4f}",
            "active": "1" if active else "0",
            "interests": ",".join(c["interests"] or []),
            "countries": ",".join(c["countries"] or []),
            "freq_cap_hour": str(c["freq_cap_hour"]),
            "pacing_mul": f"{float(c['pacing_mul']):.4f}",
        }
        pipe.hset(f"campaign:{cid}", mapping=mapping)
        pipe.set(f"pacing:{cid}", f"{float(c['pacing_mul']):.4f}")
        if active:
            for interest in c["interests"] or []:
                pipe.sadd(f"idx:interest:{interest.lower()}", cid)
            for country in c["countries"] or []:
                pipe.sadd(f"idx:country:{country.upper()}", cid)
    pipe.set("cache:generation", str(int(time.time())))
    pipe.execute()
    return active_count


def sync_once() -> int:
    with psycopg.connect(postgres_dsn()) as conn:
        campaigns = load_campaigns(conn)
    r = redis.Redis.from_url(redis_url(), decode_responses=True)
    return publish(r, campaigns)


def run(interval_sec: float | None = None) -> None:
    if interval_sec is None:
        interval_sec = float(os.getenv("CACHE_SYNC_INTERVAL_SEC", "2"))
    start_http_server(int(os.getenv("METRICS_PORT", "9102")))
    signal.signal(signal.SIGINT, _stop)
    signal.signal(signal.SIGTERM, _stop)
    log.info("cache-sync started interval=%ss", interval_sec)
    while _running:
        try:
            n = sync_once()
            CAMPAIGNS_GAUGE.set(n)
            SYNC_TOTAL.labels("ok").inc()
            log.info("synced %s active campaigns", n)
        except Exception:
            SYNC_TOTAL.labels("error").inc()
            log.exception("sync failed")
        time.sleep(interval_sec)
    log.info("cache-sync stopped")


if __name__ == "__main__":
    run()
