package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"adplatform/platform/contracts"
)

// Redis key layout (owned by cache-sync, read by adserver):
//
//	campaign:{id}                 hash of serving fields
//	idx:interest:{interest}       set of campaign ids
//	idx:country:{country}         set of campaign ids
//	pacing:{id}                   float multiplier
//	freq:{user}:{campaign}        hourly impression counter
const (
	campaignKeyPrefix = "campaign:"
	interestKeyPrefix = "idx:interest:"
	countryKeyPrefix  = "idx:country:"
	pacingKeyPrefix   = "pacing:"
	freqKeyPrefix     = "freq:"
)

// CampaignView is the serving path's read model.
type CampaignView struct {
	rdb *redis.Client
}

func NewCampaignView(rdb *redis.Client) *CampaignView {
	return &CampaignView{rdb: rdb}
}

func NewRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  20 * time.Millisecond,
		WriteTimeout: 20 * time.Millisecond,
		PoolSize:     64,
	})
}

func (v *CampaignView) Ping(ctx context.Context) error {
	return v.rdb.Ping(ctx).Err()
}

// Eligible intersects interest indexes, then filters by country index membership
// and loads campaign hashes. A Redis error fails closed for targeting.
func (v *CampaignView) Eligible(ctx context.Context, req contracts.AdRequest) ([]*Campaign, error) {
	ids, err := v.loadByInterests(ctx, req.Interests, req.Country)
	if err != nil {
		return nil, err
	}
	out := make([]*Campaign, 0, len(ids))
	for _, id := range ids {
		c, err := v.loadCampaign(ctx, id)
		if err != nil {
			return nil, err
		}
		if c == nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (v *CampaignView) loadByInterests(ctx context.Context, interests []string, country string) ([]string, error) {
	if len(interests) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(interests))
	for _, interest := range interests {
		interest = strings.TrimSpace(strings.ToLower(interest))
		if interest == "" {
			continue
		}
		keys = append(keys, interestKeyPrefix+interest)
	}
	if len(keys) == 0 {
		return nil, nil
	}

	// SUNION of interest sets, then keep campaigns also in the country set.
	pipe := v.rdb.Pipeline()
	unionCmd := pipe.SUnion(ctx, keys...)
	var countryCmd *redis.StringSliceCmd
	if country != "" {
		countryCmd = pipe.SMembers(ctx, countryKeyPrefix+strings.ToUpper(country))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("interest index: %w", err)
	}
	interestIDs, err := unionCmd.Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if countryCmd == nil {
		return interestIDs, nil
	}
	countryIDs, err := countryCmd.Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(countryIDs))
	for _, id := range countryIDs {
		allowed[id] = struct{}{}
	}
	filtered := make([]string, 0, len(interestIDs))
	for _, id := range interestIDs {
		if _, ok := allowed[id]; ok {
			filtered = append(filtered, id)
		}
	}
	return filtered, nil
}

func (v *CampaignView) loadCampaign(ctx context.Context, id string) (*Campaign, error) {
	m, err := v.rdb.HGetAll(ctx, campaignKeyPrefix+id).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, nil
	}
	bid, _ := strconv.ParseFloat(m["bid_cpm"], 64)
	pacing, err := v.rdb.Get(ctx, pacingKeyPrefix+id).Float64()
	if err == redis.Nil {
		pacing, _ = strconv.ParseFloat(m["pacing_mul"], 64)
	} else if err != nil {
		return nil, err
	}
	freqCap, _ := strconv.ParseInt(m["freq_cap_hour"], 10, 64)
	active := m["active"] == "1" || strings.EqualFold(m["active"], "true")
	return &Campaign{
		ID:          id,
		CreativeID:  m["creative_id"],
		BidCPM:      bid,
		Interests:   splitCSV(m["interests"]),
		Countries:   splitCSV(m["countries"]),
		Active:      active,
		PacingMul:   pacing,
		FreqCapHour: freqCap,
	}, nil
}

// FrequencyOK increments the hourly counter. On Redis errors it returns
// (true, true, err) so callers can fail open with a metric.
func (v *CampaignView) FrequencyOK(ctx context.Context, userID, campaignID string, capPerHour int64) (bool, bool, error) {
	key := freqKeyPrefix + userID + ":" + campaignID
	n, err := v.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, true, err
	}
	if n == 1 {
		_ = v.rdb.Expire(ctx, key, time.Hour).Err()
	}
	return n <= capPerHour, false, nil
}

// MemoryStore is an in-process CampaignStore for unit tests.
type MemoryStore struct {
	Campaigns map[string]*Campaign
	Freq      map[string]int64
	FailEligible bool
	FailFreq     bool
}

func NewMemoryStore(campaigns ...*Campaign) *MemoryStore {
	m := &MemoryStore{
		Campaigns: make(map[string]*Campaign),
		Freq:      make(map[string]int64),
	}
	for _, c := range campaigns {
		cp := *c
		m.Campaigns[c.ID] = &cp
	}
	return m
}

func (m *MemoryStore) Ping(context.Context) error { return nil }

func (m *MemoryStore) Eligible(ctx context.Context, req contracts.AdRequest) ([]*Campaign, error) {
	if m.FailEligible {
		return nil, fmt.Errorf("redis down")
	}
	interestSet := make(map[string]struct{}, len(req.Interests))
	for _, i := range req.Interests {
		interestSet[strings.ToLower(i)] = struct{}{}
	}
	out := make([]*Campaign, 0)
	for _, c := range m.Campaigns {
		if !c.Active {
			continue
		}
		match := false
		for _, i := range c.Interests {
			if _, ok := interestSet[strings.ToLower(i)]; ok {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		if !countryMatch(c.Countries, req.Country) {
			continue
		}
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}

func (m *MemoryStore) FrequencyOK(ctx context.Context, userID, campaignID string, capPerHour int64) (bool, bool, error) {
	if m.FailFreq {
		return true, true, fmt.Errorf("redis down")
	}
	key := userID + ":" + campaignID
	m.Freq[key]++
	return m.Freq[key] <= capPerHour, false, nil
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// EncodeCampaignHash is used by tests / tools that want Redis-shaped data.
func EncodeCampaignHash(c *Campaign) map[string]string {
	active := "0"
	if c.Active {
		active = "1"
	}
	b, _ := json.Marshal(c.Interests)
	_ = b
	return map[string]string{
		"creative_id":   c.CreativeID,
		"bid_cpm":       strconv.FormatFloat(c.BidCPM, 'f', 4, 64),
		"active":        active,
		"interests":     strings.Join(c.Interests, ","),
		"countries":     strings.Join(c.Countries, ","),
		"freq_cap_hour": strconv.FormatInt(c.FreqCapHour, 10),
		"pacing_mul":    strconv.FormatFloat(c.PacingMul, 'f', 4, 64),
	}
}
