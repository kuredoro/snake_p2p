package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/i582/cfmt/cmd/cfmt"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/kuredoro/snake_p2p/protocol/gather"
)

const SendEvery = time.Second

func HostAddrInfo(h host.Host) *peer.AddrInfo {
	return &peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
}

func main() {
	peerAddrFlag := flag.String("peer", "", "peer to connect to")
	gatherFlag := flag.Bool("gather", false, "should this peer announce a gather point?")
	flag.Parse()

	// Set up host
	fmt.Print("Setting up host...")
	os.Stdout.Sync()

	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		printErr("init node:", err)
		os.Exit(1)
	}
	defer h.Close()
	fmt.Println("ok")

	// Set up mDNS discovery
	if err := setupDiscovery(h); err != nil {
		printErr("setup discovery:", err)
		os.Exit(1)
	}
	fmt.Println("Now listening")

	if *peerAddrFlag != "" {
		pi, err := peer.AddrInfoFromString(*peerAddrFlag)
		if err != nil {
			printErr("parse peer p2p multiaddr:", err)
			os.Exit(1)
		}

		err = h.Connect(context.Background(), *pi)
		if err != nil {
			fmt.Printf("ERR connecting to peer %v: %v\n", pi.ID.Pretty(), err)
		}
	}

	// Set up pub/sub
	ctx := context.Background()
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		printErr("enable pubsub:", err)
		os.Exit(1)
	}

	fmt.Print("Joining the network...")
	m, err := JoinNetwork(ctx, h, ps)
	if err != nil {
		printErr("join the network:", err)
		os.Exit(1)
	}
	fmt.Println("ok")

	if *gatherFlag {
		m.CreateGatherPoint(SendEvery)
	}

	// Read from the channel and send
	for {
		msg := <-m.Messages
		fmt.Printf("GHR %v/%v %v\n", msg.CurrentPlayerCount, msg.DesiredPlayerCount, msg.ConnectTo)
		m.JoinGatherPoint(msg.ConnectTo)
	}
}

type NetworkMember struct {
	ctx      context.Context
	h        host.Host
	ps       *pubsub.PubSub
	topic    *pubsub.Topic
	sub      *pubsub.Subscription
	addrInfo *peer.AddrInfo

	joinedGatherPoints map[peer.ID]*gather.JoinService
	gatherService      *gather.GatherService
	Messages           chan *gather.GatherPointMessage
}

func JoinNetwork(ctx context.Context, h host.Host, ps *pubsub.PubSub) (*NetworkMember, error) {
	topic, err := ps.Join("snake_test")
	if err != nil {
		return nil, fmt.Errorf("join topic %q: %v", "snake_test", topic)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("subscribe to %v: %v", topic, err)
	}

	nm := &NetworkMember{
		ctx:                ctx,
		h:                  h,
		ps:                 ps,
		topic:              topic,
		sub:                sub,
		addrInfo:           HostAddrInfo(h),
		joinedGatherPoints: make(map[peer.ID]*gather.JoinService),
		Messages:           make(chan *gather.GatherPointMessage, 32),
	}

	go nm.readLoop()
	return nm, nil
}

func (nm *NetworkMember) JoinGatherPoint(pi peer.AddrInfo) error {
	if _, joined := nm.joinedGatherPoints[pi.ID]; joined {
		return nil
	}

	err := nm.h.Connect(nm.ctx, pi)
	if err != nil {
		return fmt.Errorf("join gather point: %v", err)
	}

	service, err := gather.NewJoinService(nm.ctx, nm.h, pi.ID)
	if err != nil {
		return fmt.Errorf("create join service for peer %v: %v", pi.ID.ShortString(), err)
	}

	nm.joinedGatherPoints[pi.ID] = service

	fmt.Printf("JOINED %v\n", pi.ID)

	return nil
}

func (nm *NetworkMember) CreateGatherPoint(TTL time.Duration) (err error) {
	nm.gatherService, err = gather.NewGatherService(nm.ctx, nm.h, nm.topic, SendEvery)
	if err != nil {
		return fmt.Errorf("create gather point: %v", err)
	}

	return nil
}

func (nm *NetworkMember) readLoop() {
	for {
		psMsg, err := nm.sub.Next(nm.ctx)
		if err != nil {
			printErr("receive next message:", err)
			close(nm.Messages)
			return
		}

		if psMsg.ReceivedFrom == nm.addrInfo.ID {
			continue
		}

		fmt.Printf("From %v, ReceivedFrom %v\n", psMsg.GetFrom(), psMsg.ReceivedFrom)

		msg := &gather.GatherPointMessage{}
		if err := json.Unmarshal(psMsg.Data, &msg); err != nil {
			cfmt.Printf("{{warning:}}::lightYellow|bold couldn't unmarshal %q\n", string(psMsg.Data))
			continue
		}

		nm.Messages <- msg
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
