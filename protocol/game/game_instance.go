package game

import (
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

type GameInstance struct {
	streams map[peer.ID]network.Stream

	mu sync.Mutex
}

func NewGameInstance() *GameInstance {
	return &GameInstance{
		streams: make(map[peer.ID]network.Stream),
	}
}

func (gi *GameInstance) AddPeer(s network.Stream) {
	p := s.Conn().RemotePeer()

	gi.mu.Lock()
	gi.streams[p] = s
	gi.mu.Unlock()
}

func (gi *GameInstance) RemovePeer(p peer.ID) error {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	s, exists := gi.streams[p]
	if !exists {
		return nil
	}

	err := s.Close()
	if err != nil {
		return fmt.Errorf("close game stream: %v", err)
	}

	delete(gi.streams, p)

	return nil
}

func (gi *GameInstance) Close() {
	gi.mu.Lock()
	defer gi.mu.Unlock()

	for p, s := range gi.streams {
		err := s.Close()
		if err != nil {
			log.Err(err).
				Str("peer", p.Pretty()).
				Msg("Close game stream")
		}
	}
}
