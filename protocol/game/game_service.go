package game

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
)

const ID = "/snake/game/0.1.0"

type GameService struct {
	h        host.Host
	instance *GameInstance
}

func NewGameService(h host.Host) (*GameService, error) {
	game := &GameService{
		h:        h,
		instance: NewGameInstance(),
	}

	h.SetStreamHandler(ID, game.GameHandler)

	return game, nil
}

func (g *GameService) Connect(ctx context.Context, p peer.ID) error {
	s, err := g.h.NewStream(ctx, p, ID)
	if err != nil {
		// TODO: maybe PeerError? But then how to zerolog?
		return fmt.Errorf("new game stream: %v", err)
	}

	g.instance.AddPeer(s)

	return nil
}

func (g *GameService) Disconnect(p peer.ID) {
	g.instance.RemovePeer(p)
}

func (g *GameService) GameHandler(s network.Stream) {
	p := s.Conn().RemotePeer()
	log.Info().
		Str("peer", p.Pretty()).
		Msg("New incomming game connection")

	g.instance.AddPeer(s)
}

func (g *GameService) Close() {
	g.instance.Close()
}
