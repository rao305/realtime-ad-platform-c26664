package decision

import (
	"context"

	"adplatform/contracts"
)

// Campaign is the serving-time view of a campaign (cached, not the full DB row).
type Campaign struct {
	ID         string
	CreativeID string
	BidCPM     float64  // advertiser's bid (cost per mille)
	Interests  []string // targeting: any-match
	Countries  []string // targeting: membership
	Active     bool     // paused/over-budget campaigns are inactive
	PacingMul  float64  // 0..1 budget-pacing throttle (set by the write path)
}

type Engine struct {
	view *CampaignView // hot, cached set of candidate campaigns
}

func NewEngine() *Engine { return &Engine{view: NewCampaignView()} }

// Decide runs the serving decision under the request's deadline.
func (e *Engine) Decide(ctx context.Context, req contracts.AdRequest) contracts.AdResponse {
	noAd := contracts.AdResponse{RequestID: req.RequestID, Served: false}

	// Respect the deadline: if we're already out of time, degrade now.
	if ctx.Err() != nil {
		return noAd
	}

	eligible := e.view.Eligible(req) // already-indexed candidate set
	if len(eligible) == 0 {
		return noAd
	}

	// Rank by effective value = bid * pacing throttle. Pacing lets the write
	// path slow a campaign that's spending too fast without pausing it.
	var best *Campaign
	var bestScore float64
	for i := range eligible {
		c := eligible[i]
		score := c.BidCPM * c.PacingMul
		if score > bestScore {
			bestScore, best = score, c
		}
	}
	if best == nil {
		return noAd
	}
	return contracts.AdResponse{
		RequestID:  req.RequestID,
		CampaignID: best.ID,
		CreativeID: best.CreativeID,
		Served:     true,
	}
}
