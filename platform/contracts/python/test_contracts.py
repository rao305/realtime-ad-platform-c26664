import json
from pathlib import Path

import pytest

from contracts import EVENT_SCHEMA_VERSION, validate_ad_request, validate_event


FIXTURES = Path(__file__).resolve().parents[1] / "fixtures"


def test_request_fixture_valid():
    payload = json.loads((FIXTURES / "request.json").read_text())
    validate_ad_request(payload)


def test_event_fixture_valid():
    payload = json.loads((FIXTURES / "event.json").read_text())
    validate_event(payload)
    assert payload["schema_version"] == EVENT_SCHEMA_VERSION


def test_event_rejects_bad_type():
    with pytest.raises(ValueError):
        validate_event(
            {
                "schema_version": 1,
                "event_id": "e1",
                "type": "view",
                "request_id": "r1",
                "campaign_id": "c1",
                "user_id": "u1",
                "at": "2026-07-30T00:00:00Z",
            }
        )
