package core

import (
	"github.com/kuredoro/snake_p2p/protocol/game"
	"github.com/libp2p/go-libp2p-core/peer"
)

type GameEstablished struct {
	Facilitator peer.ID
	Game        *game.GameInstance
}
