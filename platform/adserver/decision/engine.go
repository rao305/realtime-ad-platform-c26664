package decision

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"adplatform/platform/contracts"
)

// Campaign is the serving-time view of a campaign (cached, not the full DB row).
type Campaign struct {
	ID           string
	CreativeID   string
	BidCPM       float64
	Interests    []string
	Countries    []string
	Active       bool
	PacingMul    float64
	FreqCapHour  int64
}

// CampaignStore is the Redis-backed (or in-memory) candidate lookup.
type CampaignStore interface {
	Eligible(ctx context.Context, req contracts.AdRequest) ([]*Campaign, error)
	FrequencyOK(ctx context.Context, userID, campaignID string, capPerHour int64) (bool, bool, error)
	Ping(ctx context.Context) error
}

// Engine ranks eligible campaigns under a latency budget.
type Engine struct {
	store          CampaignStore
	freqCapDefault int64
	log            *slog.Logger
}

func NewEngine(store CampaignStore, freqCapDefault int64, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.Default()
	}
	if freqCapDefault <= 0 {
		freqCapDefault = 5
	}
	return &Engine{store: store, freqCapDefault: freqCapDefault, log: log}
}

// Decide runs targeting, frequency capping, and ranking.
func (e *Engine) Decide(ctx context.Context, req contracts.AdRequest) contracts.AdResponse {
	noAd := contracts.AdResponse{RequestID: req.RequestID, Served: false}

	if ctx.Err() != nil {
		noAd.Reason = "deadline_exceeded"
		DecisionsTotal.WithLabelValues("timeout").Inc()
		return noAd
	}

	eligible, err := e.store.Eligible(ctx, req)
	if err != nil {
		e.log.Warn("eligible lookup failed", "err", err, "request_id", req.RequestID)
		CacheErrorsTotal.WithLabelValues("eligible").Inc()
		noAd.Reason = "cache_unavailable"
		DecisionsTotal.WithLabelValues("cache_error").Inc()
		return noAd
	}
	if len(eligible) == 0 {
		noAd.Reason = "no_eligible"
		DecisionsTotal.WithLabelValues("no_eligible").Inc()
		return noAd
	}

	candidates := make([]*Campaign, 0, len(eligible))
	for _, c := range eligible {
		if ctx.Err() != nil {
			noAd.Reason = "deadline_exceeded"
			DecisionsTotal.WithLabelValues("timeout").Inc()
			return noAd
		}
		if c == nil || !c.Active || c.PacingMul <= 0 {
			continue
		}
		if !countryMatch(c.Countries, req.Country) {
			continue
		}
		cap := c.FreqCapHour
		if cap <= 0 {
			cap = e.freqCapDefault
		}
		ok, failOpen, ferr := e.store.FrequencyOK(ctx, req.UserID, c.ID, cap)
		if ferr != nil {
			e.log.Warn("frequency check failed; failing open", "err", ferr, "campaign", c.ID)
			FreqFailOpenTotal.Inc()
			ok = true
			_ = failOpen
		} else if failOpen {
			FreqFailOpenTotal.Inc()
		}
		if !ok {
			continue
		}
		candidates = append(candidates, c)
	}
	if len(candidates) == 0 {
		noAd.Reason = "filtered"
		DecisionsTotal.WithLabelValues("filtered").Inc()
		return noAd
	}

	best := rank(candidates)
	DecisionsTotal.WithLabelValues("served").Inc()
	return contracts.AdResponse{
		RequestID:  req.RequestID,
		CampaignID: best.ID,
		CreativeID: best.CreativeID,
		Served:     true,
		BidCPM:     best.BidCPM,
	}
}

func countryMatch(countries []string, country string) bool {
	if len(countries) == 0 {
		return true
	}
	country = strings.ToUpper(country)
	for _, c := range countries {
		if strings.ToUpper(c) == country {
			return true
		}
	}
	return false
}

// rank picks max BidCPM * PacingMul; ties break by campaign ID for determinism.
func rank(candidates []*Campaign) *Campaign {
	sort.SliceStable(candidates, func(i, j int) bool {
		si := candidates[i].BidCPM * candidates[i].PacingMul
		sj := candidates[j].BidCPM * candidates[j].PacingMul
		if si == sj {
			return candidates[i].ID < candidates[j].ID
		}
		return si > sj
	})
	return candidates[0]
}
