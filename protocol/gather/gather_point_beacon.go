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
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	ttl      time.Duration
	selfInfo peer.AddrInfo
	topic    *pubsub.Topic
}

func NewGatherPointBeacon(ctx context.Context, topic *pubsub.Topic, self peer.AddrInfo, TTL time.Duration) *GatherPointBeacon {
	localCtx, cancel := context.WithCancel(ctx)

	b := &GatherPointBeacon{
		ctx:    localCtx,
		cancel: cancel,
		done:   make(chan struct{}),

		ttl:      TTL,
		selfInfo: self,
		topic:    topic,
	}

	go b.publishLoop()

	return b
}

func (b *GatherPointBeacon) Done() <-chan struct{} {
	if b == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}

	return b.done
}

func (b *GatherPointBeacon) Close() {
	b.cancel()
	<-b.done
}

func (b *GatherPointBeacon) publishLoop() {
	timer := time.NewTimer(b.ttl)

	for {
		select {
		case <-timer.C:
			err := b.publish()
			if err != nil {
				fmt.Printf("gather service: announce: %v\n", err)
			}
			timer.Reset(b.ttl)
		case <-b.ctx.Done():
			close(b.done)
			return
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

	err = b.topic.Publish(b.ctx, msgBytes)
	if err != nil {
		return fmt.Errorf("publish gather point message: %v", err)
	}

	return nil
}
