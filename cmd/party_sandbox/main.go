package main

import (
	"context"
	"fmt"
	"os"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

func main() {

    // Set up host
    fmt.Print("Setting up host...")
    os.Stdout.Sync()

    h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
    if err != nil {
        panic(err)
    }
    defer h.Close()
    fmt.Println("ok")

    if err := setupDiscovery(h); err != nil {
        panic(err)
    }

    fmt.Println("Now listening")

    // Set up mDNS discovery
    // Set up pub/sub
    // Read from the channel
    // and send

    for {}
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
