"""Unit tests for spend store helpers (mocked DB/Redis)."""
from __future__ import annotations

import store as store_mod


class FakeCursor:
    def __init__(self, fetchone_value=None):
        self.fetchone_value = fetchone_value
        self.queries = []

    def __enter__(self):
        return self

    def __exit__(self, *args):
        return False

    def execute(self, sql, params=None):
        self.queries.append((sql, params))

    def fetchone(self):
        if callable(self.fetchone_value):
            return self.fetchone_value(len(self.queries))
        return self.fetchone_value


class FakeConn:
    def __init__(self, cursor: FakeCursor):
        self._cursor = cursor
        self.closed = False
        self.commits = 0
        self.rollbacks = 0

    def cursor(self):
        return self._cursor

    def commit(self):
        self.commits += 1

    def rollback(self):
        self.rollbacks += 1


def test_claim_event_first_writer(monkeypatch):
    cur = FakeCursor(fetchone_value=("evt-1",))
    conn = FakeConn(cur)
    monkeypatch.setattr(store_mod, "get_conn", lambda: conn)
    assert store_mod.claim_event("evt-1", "r1", "impression", "c1") is True
    assert conn.commits == 1


def test_claim_event_duplicate(monkeypatch):
    cur = FakeCursor(fetchone_value=None)
    conn = FakeConn(cur)
    monkeypatch.setattr(store_mod, "get_conn", lambda: conn)
    assert store_mod.claim_event("evt-1", "r1", "impression", "c1") is False


def test_record_spend_deactivates_at_budget(monkeypatch):
    # First fetchone: spent/budget/bid. Later INSERT RETURNING ignored.
    values = [(9.0, 10.0, 2.5)]

    def fetch(_n):
        return values[0] if values else None

    cur = FakeCursor(fetchone_value=fetch)
    conn = FakeConn(cur)
    monkeypatch.setattr(store_mod, "get_conn", lambda: conn)
    spent, budget, active = store_mod.record_spend("c1", 1.5, "evt-1")
    assert spent == 10.0
    assert budget == 10.0
    assert active is False
    assert conn.commits == 1


def test_record_impression_claims_and_spends_in_one_transaction(monkeypatch):
    class TransactionCursor(FakeCursor):
        def __init__(self):
            super().__init__()
            self.last_sql = ""

        def execute(self, sql, params=None):
            self.last_sql = sql
            self.queries.append((sql, params))

        def fetchone(self):
            if "INSERT INTO processed_events" in self.last_sql:
                return ("evt-atomic",)
            if "FOR UPDATE" in self.last_sql:
                return (1.0, 10.0, 2.5)
            return None

    cur = TransactionCursor()
    conn = FakeConn(cur)
    monkeypatch.setattr(store_mod, "get_conn", lambda: conn)

    claimed, spent, budget, active = store_mod.record_impression(
        "evt-atomic", "req-atomic", "camp-1", 0.0025
    )

    sql = "\n".join(query for query, _params in cur.queries)
    assert claimed is True
    assert spent == 1.0025
    assert budget == 10.0
    assert active is True
    assert "INSERT INTO processed_events" in sql
    assert "SELECT spent_today" in sql
    assert "INSERT INTO spend_ledger" in sql
    assert conn.commits == 1


def test_set_pacing_zero_when_exhausted(monkeypatch):
    cur = FakeCursor()
    conn = FakeConn(cur)

    class FakeRedis:
        def __init__(self):
            self.kv = {}
            self.hashes = {}

        def set(self, k, v):
            self.kv[k] = v

        def hset(self, k, mapping=None):
            self.hashes[k] = mapping

    r = FakeRedis()
    monkeypatch.setattr(store_mod, "get_conn", lambda: conn)
    monkeypatch.setattr(store_mod, "get_redis", lambda: r)
    mul = store_mod.set_pacing("c1", spent=10, budget=10, active=False)
    assert mul == 0.0
    assert r.kv["pacing:c1"] == "0.0000"
