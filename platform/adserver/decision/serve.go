package decision

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"adplatform/platform/contracts"
)

// EventPublisher is implemented by the Kafka emitter (keeps packages acyclic).
type EventPublisher interface {
	Emit(ctx context.Context, ev contracts.Event)
}

// Handler owns HTTP endpoints for the ad server.
type Handler struct {
	engine *Engine
	emit   EventPublisher
	budget time.Duration
	store  CampaignStore
	log    *slog.Logger
	ready  func(context.Context) bool
}

func NewHandler(engine *Engine, emit EventPublisher, budget time.Duration, store CampaignStore, log *slog.Logger, ready func(context.Context) bool) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{engine: engine, emit: emit, budget: budget, store: store, log: log, ready: ready}
}

// ServeAd is the front door. It enforces the latency budget and always returns JSON.
func (h *Handler) ServeAd(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req contracts.AdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := contracts.ValidateAdRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.budget)
	defer cancel()

	resp := h.engine.Decide(ctx, req)
	resp.DecisionMS = time.Since(start).Milliseconds()
	DecisionLatency.Observe(time.Since(start).Seconds())

	if resp.Served && h.emit != nil {
		ev := contracts.Event{
			SchemaVersion: contracts.EventSchemaVersion,
			EventID:       "imp:" + req.RequestID + ":" + resp.CampaignID,
			Type:          contracts.EventImpression,
			RequestID:     req.RequestID,
			CampaignID:    resp.CampaignID,
			CreativeID:    resp.CreativeID,
			UserID:        req.UserID,
			BidCPM:        resp.BidCPM,
			At:            time.Now().UTC(),
		}
		// Fire-and-forget: never block the response on Kafka.
		go h.emit.Emit(context.Background(), ev)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// TrackClick accepts an explicit click for a previously served request.
func (h *Handler) TrackClick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		RequestID  string `json:"request_id"`
		CampaignID string `json:"campaign_id"`
		CreativeID string `json:"creative_id"`
		UserID     string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.RequestID == "" || body.CampaignID == "" || body.UserID == "" {
		http.Error(w, "request_id, campaign_id, user_id required", http.StatusBadRequest)
		return
	}
	if h.emit != nil {
		ev := contracts.Event{
			SchemaVersion: contracts.EventSchemaVersion,
			EventID:       "clk:" + body.RequestID + ":" + body.CampaignID,
			Type:          contracts.EventClick,
			RequestID:     body.RequestID,
			CampaignID:    body.CampaignID,
			CreativeID:    body.CreativeID,
			UserID:        body.UserID,
			At:            time.Now().UTC(),
		}
		go h.emit.Emit(context.Background(), ev)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	if h.ready != nil && !h.ready(r.Context()) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}
