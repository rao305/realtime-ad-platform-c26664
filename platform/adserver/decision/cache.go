package decision

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"adplatform/contracts"
)

// CampaignView is the serving path's read model: candidate campaigns indexed by
// targeting, refreshed from Redis. The ad server NEVER reads PostgreSQL on the
// hot path — that would blow the budget. PostgreSQL is the write-path's truth;
// Redis is the read-path's truth, kept in sync by a refresher.
type CampaignView struct {
	rdb *redis.Client
}

func NewCampaignView() *CampaignView {
	return &CampaignView{
		rdb: redis.NewClient(&redis.Options{Addr: "redis:6379"}),
	}
}

// Eligible returns campaigns whose targeting matches the request. The candidate
// sets are precomputed per interest as Redis sets, so this is a few set reads,
// not a scan.
func (v *CampaignView) Eligible(req contracts.AdRequest) []*Campaign {
	// (Targeting index lookups elided for brevity — keyed by interest/country.)
	return v.loadByInterests(req.Interests, req.Country)
}

// FrequencyOK enforces "show campaign C to user U at most N times per hour"
// using an atomic INCR with a TTL. Returns true if the impression is allowed.
func (v *CampaignView) FrequencyOK(ctx context.Context, userID, campaignID string, capPerHour int64) bool {
	key := "freq:" + userID + ":" + campaignID
	n, err := v.rdb.Incr(ctx, key).Result()
	if err != nil {
		// Fail OPEN: if Redis is unavailable, don't block serving on the cap.
		return true
	}
	if n == 1 {
		// First impression this window — start the 1-hour expiry.
		v.rdb.Expire(ctx, key, time.Hour)
	}
	return n <= capPerHour
}
