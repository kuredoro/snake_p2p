package snake_p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"

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

type GameEstablished struct {
	Facilitator peer.ID
	// TODO: SnakeService
}

type Node struct {
	h        host.Host
	ps       *pubsub.PubSub
	topic    *pubsub.Topic
	sub      *pubsub.Subscription
	addrInfo *peer.AddrInfo
	ping     *ping.PingService

	joinedGatherPoints map[peer.ID]*gather.JoinService
	gatherService      *gather.GatherService
	GatherPoints       chan *gather.GatherPointMessage
	EstablishedGames   chan GameEstablished
	gameProxyCh        chan GameEstablished
}

func New(ctx context.Context) (*Node, error) {
	// Set up host
	fmt.Print("Setting up host...")
	os.Stdout.Sync()

	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return nil, fmt.Errorf("init libp2p host: %v", err)
	}
	fmt.Println("ok")

	// Set up mDNS discovery
	if err := setupDiscovery(h); err != nil {
		return nil, fmt.Errorf("setup discovery: %v", err)
	}
	fmt.Println("Now listening")

	// Set up pub/sub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("enable pubsub: %v", err)
	}

	fmt.Print("Joining the network...")
	topic, err := ps.Join("snake_test")
	if err != nil {
		return nil, fmt.Errorf("join topic: %v", topic)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("subscribe to %v: %v", topic, err)
	}
	fmt.Println("ok")

	n := &Node{
		h:                  h,
		ps:                 ps,
		topic:              topic,
		sub:                sub,
		addrInfo:           HostAddrInfo(h),
		ping:               ping.NewPingService(h),
		joinedGatherPoints: make(map[peer.ID]*gather.JoinService),
		GatherPoints:       make(chan *gather.GatherPointMessage, 32),
		EstablishedGames:   make(chan GameEstablished),
		gameProxyCh:        make(chan GameEstablished),
	}

	go n.readLoop()
	return n, nil
}

func (n *Node) Close() {
	fmt.Println("start 1")
	if n.gatherService != nil {
		n.gatherService.Close()
	}
	fmt.Println("end 2")
	for i, js := range n.joinedGatherPoints {
		fmt.Println("start", i)
		js.Close()
		fmt.Println("end", i)
	}

	n.h.Close()
}

func (n *Node) JoinGatherPoint(ctx context.Context, pi peer.AddrInfo) error {
	if _, joined := n.joinedGatherPoints[pi.ID]; joined {
		return nil
	}

	err := n.h.Connect(ctx, pi)
	if err != nil {
		return fmt.Errorf("join gather point: %v", err)
	}

	service, err := gather.NewJoinService(ctx, n.h, n.ping, pi.ID)
	if err != nil {
		return fmt.Errorf("create join service for peer %v: %v", pi.ID.ShortString(), err)
	}

	n.joinedGatherPoints[pi.ID] = service

	fmt.Printf("JOINED %v\n", pi.ID)

	return nil
}

func (n *Node) CreateGatherPoint(TTL time.Duration) (err error) {
	n.gatherService, err = gather.NewGatherService(n.h, n.topic, n.ping, SendEvery)
	if err != nil {
		return fmt.Errorf("create gather point: %v", err)
	}

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
			printErr("receive next message:", err)
			close(n.GatherPoints)
			return
		case psMsg := <-subCh:
			go next()

			if psMsg.ReceivedFrom == n.addrInfo.ID {
				continue
			}

			msg := &gather.GatherPointMessage{}
			if err := json.Unmarshal(psMsg.Data, &msg); err != nil {
				cfmt.Printf("{{warning:}}::lightYellow|bold couldn't unarshal %q\n", string(psMsg.Data))
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

func printErr(m string, args ...interface{}) {
	if len(args) == 0 {
		panic("printErr: no arguments passed")
	}

	err := args[len(args)-1]

	header := m
	if len(args) > 1 {
		header = fmt.Sprintf(m, args[:len(args)-1])
	}

	cfmt.Printf("{{error:}}::lightRed|bold %s %v\n", header, err)
}
