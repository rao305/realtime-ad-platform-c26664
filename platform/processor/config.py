"""Shared environment helpers for Python services."""
from __future__ import annotations

import os


def getenv(key: str, default: str = "") -> str:
    return os.getenv(key, default)


def require(key: str, default: str = "") -> str:
    value = os.getenv(key, default)
    if not value:
        raise RuntimeError(f"{key} is required")
    return value


def postgres_dsn() -> str:
    return require(
        "POSTGRES_DSN",
        "postgres://ads:ads@postgres:5432/ads?sslmode=disable",
    )


def redis_url() -> str:
    addr = getenv("REDIS_ADDR", "redis:6379")
    if addr.startswith("redis://"):
        return addr
    return f"redis://{addr}/0"


def kafka_brokers() -> list[str]:
    raw = getenv("KAFKA_BROKERS", "kafka:9092")
    return [p.strip() for p in raw.split(",") if p.strip()]


def kafka_topic() -> str:
    return getenv("KAFKA_TOPIC", "ad-events")


def kafka_dlq_topic() -> str:
    return getenv("KAFKA_DLQ_TOPIC", "ad-events-dlq")
