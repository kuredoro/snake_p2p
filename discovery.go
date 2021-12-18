package snake_p2p

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
	if pi.ID == n.h.ID() {
		return
	}

	fmt.Printf("DISCOVERED %s\n", pi.ID)
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("ERR connecting to peer %v: %v\n", pi.ID.Pretty(), err)
	}

	info := n.h.Peerstore().PeerInfo(pi.ID)
	fmt.Printf("PEER ADDR INFO %s %v\n", info.ID, info.Addrs)
}

func setupDiscovery(h host.Host) error {
	s := mdns.NewMdnsService(h, "snake_p2p", &discoveryNotifee{h})
	return s.Start()
}
