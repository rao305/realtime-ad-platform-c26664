package contracts

import (
	"fmt"
	"strings"
)

// ValidateAdRequest checks required fields for /serve.
func ValidateAdRequest(req AdRequest) error {
	if strings.TrimSpace(req.RequestID) == "" {
		return fmt.Errorf("request_id is required")
	}
	if strings.TrimSpace(req.UserID) == "" {
		return fmt.Errorf("user_id is required")
	}
	if strings.TrimSpace(req.Country) == "" {
		return fmt.Errorf("country is required")
	}
	return nil
}

// ValidateEvent checks required fields and schema version for pipeline events.
func ValidateEvent(ev Event) error {
	if ev.SchemaVersion != EventSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", ev.SchemaVersion)
	}
	if strings.TrimSpace(ev.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if ev.Type != EventImpression && ev.Type != EventClick {
		return fmt.Errorf("type must be impression or click")
	}
	if strings.TrimSpace(ev.RequestID) == "" {
		return fmt.Errorf("request_id is required")
	}
	if strings.TrimSpace(ev.CampaignID) == "" {
		return fmt.Errorf("campaign_id is required")
	}
	if strings.TrimSpace(ev.UserID) == "" {
		return fmt.Errorf("user_id is required")
	}
	if ev.At.IsZero() {
		return fmt.Errorf("at is required")
	}
	return nil
}
