package decision

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"adplatform/contracts"
)

// budget is the hard p99 deadline for a serving decision.
const budget = 30 * time.Millisecond

// ServeAd is the front door. It enforces the latency budget with a context
// deadline and ALWAYS returns a response — a real ad if the decision finishes
// in time, otherwise a graceful empty/house response.
func (e *Engine) ServeAd(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req contracts.AdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// The decision must finish within the budget. If it doesn't, ctx is
	// cancelled and Decide returns the no-ad fallback.
	ctx, cancel := context.WithTimeout(r.Context(), budget)
	defer cancel()

	resp := e.Decide(ctx, req)
	resp.DecisionMS = time.Since(start).Milliseconds()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
