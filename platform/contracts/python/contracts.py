"""Python mirror of the Go wire contracts.

Keep field names and schema_version in lockstep with platform/contracts/ad.go
and the JSON fixtures under platform/contracts/fixtures/.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

EVENT_SCHEMA_VERSION = 1
EVENT_IMPRESSION = "impression"
EVENT_CLICK = "click"


@dataclass
class AdRequest:
    request_id: str
    user_id: str
    context: str = ""
    interests: list[str] = field(default_factory=list)
    country: str = ""


@dataclass
class Event:
    schema_version: int
    event_id: str
    type: str
    request_id: str
    campaign_id: str
    user_id: str
    at: datetime
    creative_id: str = ""
    bid_cpm: float = 0.0


def validate_ad_request(payload: dict[str, Any]) -> None:
    for key in ("request_id", "user_id", "country"):
        if not str(payload.get(key, "")).strip():
            raise ValueError(f"{key} is required")


def validate_event(payload: dict[str, Any]) -> None:
    if payload.get("schema_version") != EVENT_SCHEMA_VERSION:
        raise ValueError(f"unsupported schema_version {payload.get('schema_version')}")
    for key in ("event_id", "request_id", "campaign_id", "user_id", "at"):
        if not str(payload.get(key, "")).strip():
            raise ValueError(f"{key} is required")
    if payload.get("type") not in (EVENT_IMPRESSION, EVENT_CLICK):
        raise ValueError("type must be impression or click")


def event_from_dict(payload: dict[str, Any]) -> Event:
    validate_event(payload)
    at = payload["at"]
    if isinstance(at, str):
        at = datetime.fromisoformat(at.replace("Z", "+00:00"))
    return Event(
        schema_version=int(payload["schema_version"]),
        event_id=payload["event_id"],
        type=payload["type"],
        request_id=payload["request_id"],
        campaign_id=payload["campaign_id"],
        user_id=payload["user_id"],
        at=at,
        creative_id=payload.get("creative_id", ""),
        bid_cpm=float(payload.get("bid_cpm") or 0),
    )
