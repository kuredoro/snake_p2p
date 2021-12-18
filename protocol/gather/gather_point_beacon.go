package gather

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type GatherPointBeacon struct {
	done   chan struct{}

	ttl      time.Duration
	selfInfo peer.AddrInfo
	topic    *pubsub.Topic
}

func NewGatherPointBeacon(ctx context.Context, topic *pubsub.Topic, self peer.AddrInfo, TTL time.Duration) *GatherPointBeacon {
	b := &GatherPointBeacon{
		done:   make(chan struct{}),

		ttl:      TTL,
		selfInfo: self,
		topic:    topic,
	}

	go b.publishLoop()

	return b
}

func (b *GatherPointBeacon) Close() {
    b.done <- struct{}{}
	<-b.done
}

func (b *GatherPointBeacon) publishLoop() {
	timer := time.NewTimer(b.ttl)

	for {
		select {
        case <-b.done:
			close(b.done)
			return
		case <-timer.C:
            // TODO: if timer expires we publish the message, but what
            // if during the publishing user calls Close?
            // Maybe we should propagate context here?
            // THough only if it is a concern...
			err := b.publish()
			if err != nil {
				fmt.Printf("gather service: announce: %v\n", err)
			}

            // TODO: if b.publish takes a long time to send, then
            // we will send a new message after 2*ttl seconds in the
            // worst case. We can track time publish took and adjust
            // the timer appropriately.
			timer.Reset(b.ttl)
		}
	}
}

func (b *GatherPointBeacon) publish() error {
	msg := GatherPointMessage{
		ConnectTo:          b.selfInfo,
		TTL:                b.ttl,
		DesiredPlayerCount: 3,
		CurrentPlayerCount: 0,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal gather point messasge: %v", err)
	}

    ctx, cancel := context.WithTimeout(context.Background(), b.ttl)
    defer cancel()

	err = b.topic.Publish(ctx, msgBytes)
	if err != nil {
		return fmt.Errorf("publish gather point message: %v", err)
	}

	return nil
}
