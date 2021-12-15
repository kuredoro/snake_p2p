package main

import (
	"context"
    "encoding/json"
    "flag"
	"fmt"
	"os"
    "time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

const SendEvery = time.Second

func main() {
    tagFlag := flag.String("tag", "", "a tag that will be appended to the messages published into the network")

    // Set up host
    fmt.Print("Setting up host...")
    os.Stdout.Sync()

    h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
    if err != nil {
        panic(err)
    }
    defer h.Close()
    fmt.Println("ok")

    // Set up mDNS discovery
    if err := setupDiscovery(h); err != nil {
        panic(err)
    }
    fmt.Println("Now listening")

    // Set up pub/sub
    ctx := context.Background()
    ps, err := pubsub.NewGossipSub(ctx, h)
    if err != nil {
        panic(err)
    }

    tag := *tagFlag
    if tag == "" {
        tag = h.ID().ShortString()
    }

    fmt.Print("Joining the network...")
    m, err := JoinNetwork(ctx, ps, h.ID(), tag)
    if err != nil {
        panic(err)
    }
    fmt.Println("ok")

    // Read from the channel and send
    timer := time.NewTimer(SendEvery)
    for {
        select {
        case msg := <-m.Messages:
            fmt.Printf("MSG %v %v\n", msg.Tag, msg.Timestamp)
        case <-timer.C:
            m.Publish(time.Now())
            timer.Reset(SendEvery)
        }
    }
}

type discoveryNotifee struct {
    h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
    fmt.Printf("NEW peer: %v\n", pi)
    err := n.h.Connect(context.Background(), pi)
    if err != nil {
        fmt.Printf("ERR connecting to peer %v: %v\n", pi.ID.Pretty(), err)
    }
}

func setupDiscovery(h host.Host) error {
    s := mdns.NewMdnsService(h, "snake_test", &discoveryNotifee{h})
    return s.Start()
}

type Message struct {
    Tag string
    Timestamp time.Time
}

type NetworkMember struct {
    ctx context.Context
    ps *pubsub.PubSub
    topic *pubsub.Topic
    sub *pubsub.Subscription
    selfID peer.ID
    tag string
    Messages chan *Message
}

func JoinNetwork(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, tag string) (*NetworkMember, error) {
    topic, err := ps.Join("snake_test")
    if err != nil {
        return nil, err
    }

    sub, err := topic.Subscribe()
    if err != nil {
        return nil, err
    }

    nm := &NetworkMember{
        ctx: ctx,
        ps: ps,
        topic: topic,
        sub: sub,
        selfID: selfID,
        tag: tag,
        Messages: make(chan *Message, 32),
    }

    go nm.readLoop()
    return nm, nil
}

func (nm *NetworkMember) readLoop() {
    for {
        psMsg, err := nm.sub.Next(nm.ctx)
        if err != nil {
            close(nm.Messages)
            return
        }

        if psMsg.ReceivedFrom == nm.selfID {
            continue
        }

        msg := &Message{}
        if err := json.Unmarshal(psMsg.Data, &msg); err != nil {
            continue
        }

        nm.Messages <- msg
    }
}

func (nm *NetworkMember) Publish(timestamp time.Time) error {
    msg := Message{
        Tag: nm.tag,
        Timestamp: timestamp,
    }

    msgBytes, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    return nm.topic.Publish(nm.ctx, msgBytes)
}
