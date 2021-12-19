package snake_p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/rs/zerolog/log"

	"github.com/kuredoro/snake_p2p/protocol/game"
	"github.com/kuredoro/snake_p2p/protocol/gather"
)

const SendEvery = time.Second

// TODO: move to utility package
func HostAddrInfo(h host.Host) *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
}

type Node struct {
	h        host.Host
	ps       *pubsub.PubSub
	topic    *pubsub.Topic
	sub      *pubsub.Subscription
	addrInfo *peer.AddrInfo
	ping     *ping.PingService
	game     *game.GameService

	joinedGatherPoints            map[peer.ID]*gather.JoinService
	gatherService                 *gather.GatherService
	GatherPoints                  chan *gather.GatherPointMessage
	EstablishedGames, gameProxyCh chan game.GameEstablished
}

func New(ctx context.Context) (*Node, error) {
	// Set up host
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return nil, fmt.Errorf("init libp2p host: %v", err)
	}
	log.Info().Msg("Initialized libp2p host")

	// Set up mDNS discovery
	if err := setupDiscovery(h); err != nil {
		return nil, fmt.Errorf("setup discovery: %v", err)
	}
	log.Info().Msg("Discovery set up")

	// Set up pub/sub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("enable pubsub: %v", err)
	}

	topic, err := ps.Join("snake_test")
	if err != nil {
		return nil, fmt.Errorf("join topic: %v", topic)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("subscribe to %v: %v", topic, err)
	}

	log.Info().Msg("Joined the pub/sub network")

	n := &Node{
		h:                  h,
		ps:                 ps,
		topic:              topic,
		sub:                sub,
		addrInfo:           HostAddrInfo(h),
		ping:               ping.NewPingService(h),
		game:               game.NewGameService(h),
		joinedGatherPoints: make(map[peer.ID]*gather.JoinService),
		GatherPoints:       make(chan *gather.GatherPointMessage, 32),
		EstablishedGames:   make(chan game.GameEstablished),
		gameProxyCh:        make(chan game.GameEstablished),
	}

	go n.readLoop()
	return n, nil
}

func (n *Node) Close() {
	if n.gatherService != nil {
		log.Debug().Msg("Closing gathering service")
		n.gatherService.Close()
	}
	for i, js := range n.joinedGatherPoints {
		log.Debug().
			Str("facilitator", i.Pretty()).
			Msg("Closing join service")
		js.Close()
	}

	log.Debug().Msg("Closing libp2p host")
	err := n.h.Close()
	if err != nil {
		log.Err(err).Msg("Close libp2p host")
	}

	log.Info().Msg("Snake node closed")
}

func (n *Node) JoinGatherPoint(ctx context.Context, pi peer.AddrInfo) error {
	if _, joined := n.joinedGatherPoints[pi.ID]; joined {
		return nil
	}

	err := n.h.Connect(ctx, pi)
	if err != nil {
		return fmt.Errorf("join gather point: %v", err)
	}

	service, err := gather.NewJoinService(ctx, n.h, n.game, n.ping, pi.ID, n.gameProxyCh)
	if err != nil {
		return fmt.Errorf("create join service for peer %v: %v", pi.ID.ShortString(), err)
	}

	n.joinedGatherPoints[pi.ID] = service

	log.Info().
		Str("facilitator", pi.ID.Pretty()).
		Msg("New join service")

	return nil
}

func (n *Node) CreateGatherPoint(playerCount int, TTL time.Duration) (err error) {
	n.gatherService, err = gather.NewGatherService(n.h, n.topic, n.game, n.ping, playerCount, TTL, n.gameProxyCh)
	if err != nil {
		return fmt.Errorf("create gather point: %v", err)
	}

	log.Info().Msg("Created gather point")

	return nil
}

func (n *Node) readLoop() {
	subCh := make(chan *pubsub.Message)
	defer close(subCh)

	errCh := make(chan error)
	defer close(errCh)

	next := func() {
		msg, err := n.sub.Next(context.TODO())
		if err != nil {
			errCh <- err
			return
		}

		subCh <- msg
	}

	go next()

	for {
		select {
		// TODO: done channel to close this goroutine
		case err := <-errCh:
			log.Err(err).Msg("Receive pub/sub message")
			close(n.GatherPoints)
			return
		case psMsg := <-subCh:
			go next()

			if psMsg.ReceivedFrom == n.addrInfo.ID {
				continue
			}

			msg := &gather.GatherPointMessage{}
			if err := json.Unmarshal(psMsg.Data, &msg); err != nil {
				log.Err(err).
					Str("from", psMsg.GetFrom().Pretty()).
					Str("text", fmt.Sprintf("%q", psMsg.String())).
					Msg("Unmarshal topic message")
				continue
			}

			n.GatherPoints <- msg
		case info := <-n.gameProxyCh:
			if n.gatherService != nil {
				n.gatherService.Close()
			}

			for _, s := range n.joinedGatherPoints {
				s.Close()
			}

			n.joinedGatherPoints = make(map[peer.ID]*gather.JoinService)

			n.EstablishedGames <- info
		}
	}
}
