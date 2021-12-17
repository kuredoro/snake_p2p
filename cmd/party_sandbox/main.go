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
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "github.com/i582/cfmt/cmd/cfmt"
)

const SendEvery = time.Second

func HostAddrInfo(h host.Host) *peer.AddrInfo {
    return &peer.AddrInfo{
        ID: h.ID(),
        Addrs: h.Addrs(),
    }
}

func main() {

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

    // Set up pub/sub
    ctx := context.Background()
    ps, err := pubsub.NewGossipSub(ctx, h)
    if err != nil {
        printErr("enable pubsub:", err)
        os.Exit(1)
    }

    fmt.Print("Joining the network...")
    m, err := JoinNetwork(ctx, ps, HostAddrInfo(h))
    if err != nil {
        printErr("join the network:", err)
        os.Exit(1)
    }
    fmt.Println("ok")

    // Read from the channel and send
    timer := time.NewTimer(SendEvery)
    for {
        select {
        case msg := <-m.Messages:
            fmt.Printf("GHR %v/%v %v\n", msg.CurrentPlayerCount, msg.DesiredPlayerCount, msg.ConnectTo)
        case <-timer.C:
            m.Publish(time.Now())
            timer.Reset(SendEvery)
        }
    }
}

type GatherPointMessage struct {
    ConnectTo *peer.AddrInfo
    TTL time.Duration
    DesiredPlayerCount uint
    CurrentPlayerCount uint
}

type NetworkMember struct {
    ctx context.Context
    ps *pubsub.PubSub
    topic *pubsub.Topic
    sub *pubsub.Subscription
    addrInfo *peer.AddrInfo
    Messages chan *GatherPointMessage
}

func JoinNetwork(ctx context.Context, ps *pubsub.PubSub, selfInfo *peer.AddrInfo) (*NetworkMember, error) {
    topic, err := ps.Join("snake_test")
    if err != nil {
        return nil, fmt.Errorf("join topic %q: %v", "snake_test", topic)
    }

    sub, err := topic.Subscribe()
    if err != nil {
        return nil, fmt.Errorf("subscribe to %v: %v", topic, err)
    }

    nm := &NetworkMember{
        ctx: ctx,
        ps: ps,
        topic: topic,
        sub: sub,
        addrInfo: selfInfo,
        Messages: make(chan *GatherPointMessage, 32),
    }

    go nm.readLoop()
    return nm, nil
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

        msg := &GatherPointMessage{}
        if err := json.Unmarshal(psMsg.Data, &msg); err != nil {
            cfmt.Printf("{{warning:}}::lightYellow|bold couldn't unmarshal %q\n", string(psMsg.Data))
            continue
        }

        nm.Messages <- msg
    }
}

func (nm *NetworkMember) Publish(timestamp time.Time) error {
    msg := GatherPointMessage{
        ConnectTo: nm.addrInfo,
        TTL: time.Minute,
        DesiredPlayerCount: 3,
        CurrentPlayerCount: 0,
    }

    msgBytes, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("marshal: %v", err)
    }

    err = nm.topic.Publish(nm.ctx, msgBytes)
    if err != nil {
        return fmt.Errorf("publish: %v", err)
    }

    return nil
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
