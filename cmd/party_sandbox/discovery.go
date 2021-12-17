package main

import (
    "context"
    "fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type discoveryNotifee struct {
    h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
    fmt.Printf("NEW peer: %v\n", pi)
    err := n.h.Connect(context.Background(), pi)
    if err != nil {
        fmt.Printf("ERR connecting to peer %v: %v\n", pi.ID.Pretty(), err)
    }

    fmt.Printf("PEER ADDR INFO: %v\n", n.h.Peerstore().PeerInfo(pi.ID))

}

func setupDiscovery(h host.Host) error {
    s := mdns.NewMdnsService(h, "snake_test", &discoveryNotifee{h})
    return s.Start()
}

