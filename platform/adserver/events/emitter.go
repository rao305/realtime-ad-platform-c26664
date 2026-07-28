package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
	"adplatform/contracts"
)

// Emitter publishes serving events to Kafka. Producing is asynchronous: the
// serving response is sent to the user FIRST, then the event is emitted, so the
// write path never adds to the serving latency.
type Emitter struct {
	w *kafka.Writer
}

func NewEmitter() *Emitter {
	return &Emitter{
		w: &kafka.Writer{
			Addr:     kafka.TCP("kafka:9092"),
			Topic:    "ad-events",
			Balancer: &kafka.Hash{}, // partition by message key
			Async:    true,          // do not block on broker acks
		},
	}
}

// Emit publishes one event. The Kafka message KEY is the campaign id, so all
// events for a campaign land on the same partition — giving per-campaign order
// and letting the stream processor aggregate a campaign on one consumer.
func (e *Emitter) Emit(ctx context.Context, ev contracts.Event) {
	body, _ := json.Marshal(ev)
	err := e.w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(ev.CampaignID),
		Value: body,
	})
	if err != nil {
		// At-least-once: log + let retto/redelivery handle it. We never fail
		// the user's request because an event failed to enqueue.
		log.Printf("emit failed for %s: %v", ev.RequestID, err)
	}
}
