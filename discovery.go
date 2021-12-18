package snake_p2p

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"github.com/rs/zerolog/log"
)

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.h.ID() {
		return
	}

	log.Debug().Str("id", pi.ID.Pretty()).Msg("mDNS discovered peer")

	// TODO: log only if new connection
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		log.Err(err).Str("id", pi.ID.Pretty()).Msg("Connect to peer")
	}
}

func setupDiscovery(h host.Host) error {
	s := mdns.NewMdnsService(h, "snake_p2p", &discoveryNotifee{h})
	return s.Start()
}
