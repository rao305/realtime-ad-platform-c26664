"""PostgreSQL access for the write path: durable spend accounting + idempotency.

Budget invariant: spent_today never exceeds daily_budget, and a campaign flips
inactive the moment it's exhausted. Enforced inside a single transaction so
concurrent processors can't oversell.
"""
from __future__ import annotations

import logging
from typing import Any

import psycopg
import redis

from config import postgres_dsn, redis_url

log = logging.getLogger(__name__)

_conn = None
_redis = None


def get_conn():
    global _conn
    if _conn is None or _conn.closed:
        _conn = psycopg.connect(postgres_dsn())
    return _conn


def get_redis():
    global _redis
    if _redis is None:
        _redis = redis.Redis.from_url(redis_url(), decode_responses=True)
    return _redis


def already_seen(event_id: str) -> bool:
    conn = get_conn()
    with conn.cursor() as cur:
        cur.execute("SELECT 1 FROM processed_events WHERE event_id = %s", (event_id,))
        return cur.fetchone() is not None


def claim_event(event_id: str, request_id: str, event_type: str, campaign_id: str) -> bool:
    """Insert claim row. Returns True if this worker owns the event (first writer)."""
    conn = get_conn()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO processed_events (event_id, request_id, event_type, campaign_id)
                VALUES (%s, %s, %s, %s)
                ON CONFLICT (event_id) DO NOTHING
                RETURNING event_id
                """,
                (event_id, request_id, event_type, campaign_id),
            )
            row = cur.fetchone()
        conn.commit()
        return row is not None
    except Exception:
        conn.rollback()
        raise


def record_impression(
    event_id: str,
    request_id: str,
    campaign_id: str,
    amount: float,
) -> tuple[bool, float, float, bool]:
    """Claim and bill an impression in one transaction.

    Keeping the idempotency claim, campaign lock, spend update, and ledger write
    atomic removes the crash gap where an event could be claimed but not billed.
    """
    conn = get_conn()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO processed_events (
                    event_id, request_id, event_type, campaign_id
                )
                VALUES (%s, %s, 'impression', %s)
                ON CONFLICT (event_id) DO NOTHING
                RETURNING event_id
                """,
                (event_id, request_id, campaign_id),
            )
            if cur.fetchone() is None:
                conn.rollback()
                return False, 0.0, 0.0, False

            cur.execute(
                "SELECT spent_today, daily_budget, bid_cpm FROM campaigns "
                "WHERE id = %s FOR UPDATE",
                (campaign_id,),
            )
            row = cur.fetchone()
            if row is None:
                raise ValueError(f"unknown campaign {campaign_id}")

            spent, budget, _bid = row
            old_total = float(spent)
            budget_total = float(budget)
            new_total = min(old_total + float(amount), budget_total)
            charged = new_total - old_total
            active = new_total < budget_total

            cur.execute(
                """
                UPDATE campaigns
                SET spent_today = %s, active = %s, updated_at = NOW()
                WHERE id = %s
                """,
                (new_total, active, campaign_id),
            )
            cur.execute(
                """
                INSERT INTO spend_ledger (event_id, campaign_id, amount)
                VALUES (%s, %s, %s)
                """,
                (event_id, campaign_id, charged),
            )
        conn.commit()
        return True, new_total, budget_total, active
    except Exception:
        conn.rollback()
        raise


def record_spend(campaign_id: str, amount: float, event_id: str) -> tuple[float, float, bool]:
    """Add spend atomically; deactivate when budget is hit.

    Returns (spent_today, daily_budget, active).
    """
    conn = get_conn()
    try:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT spent_today, daily_budget, bid_cpm FROM campaigns "
                "WHERE id = %s FOR UPDATE",
                (campaign_id,),
            )
            row = cur.fetchone()
            if row is None:
                raise ValueError(f"unknown campaign {campaign_id}")
            spent, budget, _bid = row
            new_total = float(spent) + float(amount)
            active = new_total < float(budget)
            if new_total > float(budget):
                new_total = float(budget)
                active = False
            cur.execute(
                """
                UPDATE campaigns
                SET spent_today = %s, active = %s, updated_at = NOW()
                WHERE id = %s
                """,
                (new_total, active, campaign_id),
            )
            cur.execute(
                """
                INSERT INTO spend_ledger (event_id, campaign_id, amount)
                VALUES (%s, %s, %s)
                ON CONFLICT (event_id) DO NOTHING
                """,
                (event_id, campaign_id, amount),
            )
        conn.commit()
        return new_total, float(budget), active
    except Exception:
        conn.rollback()
        raise


def set_pacing(campaign_id: str, spent: float, budget: float, active: bool) -> float:
    """Write pacing multiplier to Redis and Postgres.

    Linear remaining-budget pacing: as spend approaches budget, throttle toward 0.
    """
    if budget <= 0 or not active:
        mul = 0.0
    else:
        remaining = max(budget - spent, 0.0) / budget
        mul = max(min(remaining * 1.1, 1.0), 0.0)
        if remaining <= 0:
            mul = 0.0

    conn = get_conn()
    try:
        with conn.cursor() as cur:
            cur.execute(
                "UPDATE campaigns SET pacing_mul = %s, active = %s, updated_at = NOW() WHERE id = %s",
                (mul, active and mul > 0, campaign_id),
            )
        conn.commit()
    except Exception:
        conn.rollback()
        raise

    try:
        r = get_redis()
        r.set(f"pacing:{campaign_id}", f"{mul:.4f}")
        r.hset(
            f"campaign:{campaign_id}",
            mapping={
                "active": "1" if (active and mul > 0) else "0",
                "pacing_mul": f"{mul:.4f}",
            },
        )
    except Exception as exc:
        # Spend is durable; pacing feedback is best-effort and repaired by cache-sync.
        log.warning("failed to push pacing to redis for %s: %s", campaign_id, exc)
    return mul


def insert_analytics_event(ev: dict[str, Any]) -> None:
    conn = get_conn()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO analytics_events (
                    event_id, schema_version, type, request_id, campaign_id,
                    creative_id, user_id, bid_cpm, at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (event_id) DO NOTHING
                """,
                (
                    ev["event_id"],
                    ev.get("schema_version", 1),
                    ev["type"],
                    ev["request_id"],
                    ev["campaign_id"],
                    ev.get("creative_id", ""),
                    ev["user_id"],
                    ev.get("bid_cpm") or 0,
                    ev["at"],
                ),
            )
        conn.commit()
    except Exception:
        conn.rollback()
        raise
