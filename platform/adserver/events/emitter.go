package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/segmentio/kafka-go"

	"adplatform/platform/contracts"
)

var EventsEmittedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "adserver_events_emitted_total",
	Help: "Kafka events enqueued by type and result",
}, []string{"type", "result"})

// Emitter publishes serving events to Kafka asynchronously so the write path
// never sits on the response-critical path.
type Emitter struct {
	w     *kafka.Writer
	log   *slog.Logger
	topic string
}

func NewEmitter(brokers []string, topic string, log *slog.Logger) *Emitter {
	if log == nil {
		log = slog.Default()
	}
	return &Emitter{
		topic: topic,
		log:   log,
		w: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			Async:        true,
			BatchTimeout: 5 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
		},
	}
}

// Emit publishes one event. The Kafka key is the campaign id so all events for
// a campaign share a partition (per-campaign order for the spend aggregator).
func (e *Emitter) Emit(ctx context.Context, ev contracts.Event) {
	if err := contracts.ValidateEvent(ev); err != nil {
		e.log.Error("rejecting invalid event", "err", err)
		EventsEmittedTotal.WithLabelValues(ev.Type, "invalid").Inc()
		return
	}
	body, err := json.Marshal(ev)
	if err != nil {
		EventsEmittedTotal.WithLabelValues(ev.Type, "error").Inc()
		return
	}
	err = e.w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(ev.CampaignID),
		Value: body,
		Time:  ev.At,
	})
	if err != nil {
		// Never fail the user's request because an event failed to enqueue.
		e.log.Error("emit failed", "event_id", ev.EventID, "err", err)
		EventsEmittedTotal.WithLabelValues(ev.Type, "error").Inc()
		return
	}
	EventsEmittedTotal.WithLabelValues(ev.Type, "ok").Inc()
}

func (e *Emitter) Close() error {
	if e.w == nil {
		return nil
	}
	return e.w.Close()
}
