package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"adplatform/platform/contracts"
)

func fixturePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "fixtures", name)
}

func TestAdRequestFixtureRoundTrip(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("request.json"))
	if err != nil {
		t.Fatal(err)
	}
	var req contracts.AdRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	if err := contracts.ValidateAdRequest(req); err != nil {
		t.Fatalf("fixture request invalid: %v", err)
	}
	if req.RequestID != "req-fixture-1" || req.UserID != "user-42" {
		t.Fatalf("unexpected request: %+v", req)
	}
}

func TestEventFixtureRoundTrip(t *testing.T) {
	raw, err := os.ReadFile(fixturePath("event.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ev contracts.Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	if err := contracts.ValidateEvent(ev); err != nil {
		t.Fatalf("fixture event invalid: %v", err)
	}
	if ev.Type != contracts.EventImpression {
		t.Fatalf("expected impression, got %s", ev.Type)
	}
}

func TestValidateAdRequestRejectsMissingFields(t *testing.T) {
	err := contracts.ValidateAdRequest(contracts.AdRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateEventRejectsBadType(t *testing.T) {
	ev := contracts.Event{
		SchemaVersion: contracts.EventSchemaVersion,
		EventID:       "e1",
		Type:          "view",
		RequestID:     "r1",
		CampaignID:    "c1",
		UserID:        "u1",
		At:            time.Now().UTC(),
	}
	if err := contracts.ValidateEvent(ev); err == nil {
		t.Fatal("expected type validation error")
	}
}
