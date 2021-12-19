package core

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

type PeerError struct {
	Peer peer.ID
	Err  error
}

func (e *PeerError) Error() string {
	return fmt.Sprintf("peer %s: %v", e.Peer.Pretty(), e.Err)
}

func (e *PeerError) Unwrap() error {
	return e.Err
}
