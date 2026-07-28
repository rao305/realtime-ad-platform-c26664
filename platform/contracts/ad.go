// Package contracts defines the wire types shared across the serving path.
// Keeping them in one place keeps the ad server, cache, and event emitter in
// agreement about shapes.
package contracts

import "time"

// AdRequest is what the front door receives for every ad slot to fill.
type AdRequest struct {
	RequestID string   `json:"request_id"`
	UserID    string   `json:"user_id"`
	Context   string   `json:"context"`    // e.g. subreddit / page context
	Interests []string `json:"interests"`  // targeting signals
	Country   string   `json:"country"`
}

// AdResponse is the decision returned to the caller.
type AdResponse struct {
	RequestID  string `json:"request_id"`
	CampaignID string `json:"campaign_id"` // "" when no ad is served (house/empty)
	CreativeID string `json:"creative_id"`
	Served     bool   `json:"served"`
	DecisionMS int64  `json:"decision_ms"` // observed decision latency
}

// Event is one thing that happened, destined for the Kafka write path.
type Event struct {
	Type       string    `json:"type"`        // "impression" | "click"
	RequestID  string    `json:"request_id"`
	CampaignID string    `json:"campaign_id"`
	UserID     string    `json:"user_id"`
	At         time.Time `json:"at"`
}
