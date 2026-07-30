// Package contracts defines the versioned wire types shared across the
// serving path, event pipeline, and analytics sinks.
package contracts

import "time"

// EventSchemaVersion is stamped on every event so consumers can reject or
// migrate older payloads safely.
const EventSchemaVersion = 1

// Event type values.
const (
	EventImpression = "impression"
	EventClick      = "click"
)

// AdRequest is what the front door receives for every ad slot to fill.
type AdRequest struct {
	RequestID string   `json:"request_id"`
	UserID    string   `json:"user_id"`
	Context   string   `json:"context"`   // e.g. subreddit / page context
	Interests []string `json:"interests"` // targeting signals
	Country   string   `json:"country"`
}

// AdResponse is the decision returned to the caller.
type AdResponse struct {
	RequestID  string  `json:"request_id"`
	CampaignID string  `json:"campaign_id"` // "" when no ad is served (house/empty)
	CreativeID string  `json:"creative_id"`
	Served     bool    `json:"served"`
	DecisionMS int64   `json:"decision_ms"` // observed decision latency
	Reason     string  `json:"reason,omitempty"`
	BidCPM     float64 `json:"bid_cpm,omitempty"` // winning bid; also stamped on impression events
}

// Event is one thing that happened, destined for the Kafka write path.
// EventID is stable across retries so consumers can claim it once.
type Event struct {
	SchemaVersion int       `json:"schema_version"`
	EventID       string    `json:"event_id"`
	Type          string    `json:"type"` // "impression" | "click"
	RequestID     string    `json:"request_id"`
	CampaignID    string    `json:"campaign_id"`
	CreativeID    string    `json:"creative_id,omitempty"`
	UserID        string    `json:"user_id"`
	BidCPM        float64   `json:"bid_cpm,omitempty"`
	At            time.Time `json:"at"`
}
