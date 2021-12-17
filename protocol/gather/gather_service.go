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
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"

	"github.com/kuredoro/snake_p2p/protocol/heartbeat"
)

type GatherService struct {
	ctx    context.Context
	cancel func()

	h       host.Host
	streams map[peer.ID]network.Stream
	topic   *pubsub.Topic
	ttl     time.Duration

	mesh map[peer.ID]map[peer.ID]struct{}

	ping             *ping.PingService
	conns            map[peer.ID]*heartbeat.HeartbeatService
	localConnUpdates chan heartbeat.PeerStatus
}

func NewGatherService(ctx context.Context, h host.Host, topic *pubsub.Topic, ping *ping.PingService, TTL time.Duration) (*GatherService, error) {
	localCtx, cancel := context.WithCancel(ctx)

	gs := &GatherService{
		ctx:    localCtx,
		cancel: cancel,

		h:       h,
		streams: make(map[peer.ID]network.Stream),
		topic:   topic,
		ttl:     TTL,

		mesh: make(map[peer.ID]map[peer.ID]struct{}),

		ping:             ping,
		conns:            make(map[peer.ID]*heartbeat.HeartbeatService),
		localConnUpdates: make(chan heartbeat.PeerStatus),
	}

	h.SetStreamHandler(ID, gs.GatherHandler)

	go gs.publishLoop()
	go gs.monitorLoop()

	return gs, nil
}

func (gs *GatherService) GatherHandler(stream network.Stream) {
	fmt.Printf("PEER JOINED %v\n", stream.Conn().RemotePeer())

	hb, err := heartbeat.NewHeartbeat(gs.ctx, gs.ping, stream.Conn().RemotePeer(), gs.localConnUpdates)
	if err != nil {
		panic(err)
	}

	gs.conns[stream.Conn().RemotePeer()] = hb
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
func (gs *GatherService) monitorLoop() {
	for {
		select {
		case <-gs.ctx.Done():
			return
		case peerStatus := <-gs.localConnUpdates:
			switch peerStatus.Alive {
			case true:
				// TODO: rename ID to Peer
				if _, exists := gs.mesh[gs.h.ID()]; !exists {
					gs.mesh[gs.h.ID()] = make(map[peer.ID]struct{})
				}

				if _, exists := gs.mesh[peerStatus.ID]; !exists {
					gs.mesh[peerStatus.ID] = make(map[peer.ID]struct{})
				}

				gs.mesh[gs.h.ID()][peerStatus.ID] = struct{}{}
				gs.mesh[peerStatus.ID][gs.h.ID()] = struct{}{}
				fmt.Printf("PEER ALIVE %v\n", peerStatus.ID)
			case false:
				delete(gs.mesh[gs.h.ID()], peerStatus.ID)
				delete(gs.mesh[peerStatus.ID], gs.h.ID())
				fmt.Printf("PEER DEAD %v\n", peerStatus.ID)
			}
		}
	}
}
