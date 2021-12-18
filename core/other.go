package core

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

type GameEstablished struct {
	Facilitator peer.ID
	// TODO: SnakeService
}
