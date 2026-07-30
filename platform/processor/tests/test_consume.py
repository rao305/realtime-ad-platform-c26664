"""Tests for event processing / idempotency without Kafka."""
from __future__ import annotations

import consume


def test_impression_cost_uses_bid():
    assert consume.impression_cost({"bid_cpm": 2.5}) == 0.0025


def test_process_event_skips_duplicates(monkeypatch):
    spend_calls = []

    def record(*args, **kwargs):
        spend_calls.append(args)
        return (False, 0, 0, False)

    monkeypatch.setattr(consume, "record_impression", record)
    monkeypatch.setattr(
        consume,
        "claim_event",
        lambda *_a, **_k: (_ for _ in ()).throw(
            AssertionError("impressions must use the atomic record path")
        ),
    )
    monkeypatch.setattr(consume, "set_pacing", lambda *a, **k: 1.0)

    consume.process_event(
        {
            "schema_version": 1,
            "event_id": "e1",
            "type": "impression",
            "request_id": "r1",
            "campaign_id": "c1",
            "user_id": "u1",
            "bid_cpm": 2.5,
            "at": "2026-07-30T00:00:00Z",
        }
    )
    assert len(spend_calls) == 1


def test_process_event_records_impression(monkeypatch):
    seen = {}

    monkeypatch.setattr(
        consume,
        "record_impression",
        lambda eid, rid, cid, amount: seen.update(
            campaign=cid, amount=amount, event=eid, request=rid
        )
        or (True, 0.0025, 100.0, True),
    )
    monkeypatch.setattr(consume, "set_pacing", lambda *a, **k: 1.0)

    consume.process_event(
        {
            "schema_version": 1,
            "event_id": "e1",
            "type": "impression",
            "request_id": "r1",
            "campaign_id": "c1",
            "user_id": "u1",
            "bid_cpm": 2.5,
            "at": "2026-07-30T00:00:00Z",
        }
    )
    assert seen["amount"] == 0.0025
    assert seen["campaign"] == "c1"
