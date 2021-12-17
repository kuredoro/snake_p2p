package gather

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type GatherService struct {
	ctx    context.Context
	cancel func()

	h       host.Host
	streams map[peer.ID]network.Stream
	topic   *pubsub.Topic
	ttl     time.Duration

	mesh map[peer.ID]map[peer.ID]struct{}
}

func NewGatherService(ctx context.Context, h host.Host, topic *pubsub.Topic, TTL time.Duration) (*GatherService, error) {
	localCtx, cancel := context.WithCancel(ctx)

	gs := &GatherService{
		ctx:    localCtx,
		cancel: cancel,

		h:       h,
		streams: make(map[peer.ID]network.Stream),
		topic:   topic,
		ttl:     TTL,

		mesh: make(map[peer.ID]map[peer.ID]struct{}),
	}

	h.SetStreamHandler(ID, gs.GatherHandler)

	go gs.publishLoop()

	return gs, nil
}

func (gs *GatherService) GatherHandler(stream network.Stream) {
	fmt.Printf("PEER JOINED %v\n", stream.Conn().RemotePeer())

	// TODO; set only after establishing Game protocol
	gs.mesh[gs.h.ID()][stream.Conn().RemotePeer()] = struct{}{}
	gs.mesh[stream.Conn().RemotePeer()][gs.h.ID()] = struct{}{}
}

func (gs *GatherService) publishLoop() {
	timer := time.NewTimer(gs.ttl)

	for {
		select {
		case <-timer.C:
			err := gs.publish()
			if err != nil {
				fmt.Printf("gather service: announce: %v\n", err)
			}
			timer.Reset(gs.ttl)
		case <-gs.ctx.Done():
			return
		}
	}
}

func (gs *GatherService) publish() error {
	selfInfo := peer.AddrInfo{
		ID:    gs.h.ID(),
		Addrs: gs.h.Addrs(),
	}

	msg := GatherPointMessage{
		ConnectTo:          selfInfo,
		TTL:                time.Minute,
		DesiredPlayerCount: 3,
		CurrentPlayerCount: 0,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal gather point messasge: %v", err)
	}

	err = gs.topic.Publish(gs.ctx, msgBytes)
	if err != nil {
		return fmt.Errorf("publish gather point message: %v", err)
	}

	return nil
}
