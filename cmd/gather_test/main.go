package main

import (
	"context"
	"flag"
	"os"
	//	"os/signal"
	//"syscall"
	"time"

	snake "github.com/kuredoro/snake_p2p"
	"github.com/kuredoro/snake_p2p/core"
	"github.com/kuredoro/snake_p2p/protocol/gather"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	gatherFlag := flag.Int("gather", 0, "create gather point for N players")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ctx := context.Background()
	h, err := snake.New(ctx)
	if err != nil {
		log.Err(err).Msg("New node")
		os.Exit(1)
	}

	log.Info().Msg("Node initialized")

	if *gatherFlag != 0 {
		err := h.CreateGatherPoint(*gatherFlag, time.Second)
		if err != nil {
			log.Err(err).Msg("New gather point")
		}
	}

	sigCh := make(chan os.Signal, 1)
	// signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	gatherPoints := make(map[peer.ID]*gather.GatherPointMessage)
	for {
		select {
		case info := <-h.EstablishedGames:
			log.Info().
				Str("facilitator", info.Facilitator.Pretty()).
				Int("peer_count", info.Game.PeerCount()).
				Msg("Game established")

			gi := info.Game
			seed := gi.Run()

			log.Info().Int64("seed", seed).Msg("Got new seed")

			for i := 0; i < 3; i++ {
				err := gi.SendMove(core.Up)
				if err != nil {
					log.Err(err).Msg("Test move")
				}

				log.Info().Msg("Sent move")

				e, ok := <-gi.IncommingMoves()
				switch e := e.(type) {
				case core.PlayerMoves:
					if !ok {
						log.Info().Msg("GameInstance closed incoming moves")
					}

					log.Info().Msgf("Incoming message %#v", e.Moves)

					for peer, dir := range e.Moves {
						log.Info().
							Str("peer", peer.Pretty()).
							Int("dir", int(dir)).
							Msg("Player moved")
					}
				case peer.ID:
					log.Info().Str("player", e.Pretty()).Msg("Player disconnected")
				}
			}

			os.Exit(0)
		case msg := <-h.GatherPoints:
			if _, exists := gatherPoints[msg.ConnectTo.ID]; exists {
				continue
			}

			log.Info().
				Str("facilitator", msg.ConnectTo.ID.Pretty()).
				Uint("desired_player_count", msg.DesiredPlayerCount).
				Msg("Found new gather point")

			gatherPoints[msg.ConnectTo.ID] = msg

			err := h.JoinGatherPoint(ctx, msg.ConnectTo)
			if err != nil {
				log.Err(err).Msg("Join gather point")
			}
		case <-sigCh:
			h.Close()
			return
		}
	}
}
