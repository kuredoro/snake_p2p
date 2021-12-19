package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	snake "github.com/kuredoro/snake_p2p"
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
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Read from the channel and send
	for {
		select {
		case info := <-h.EstablishedGames:
			log.Info().
				Str("facilitator", info.Facilitator.Pretty()).
				Msg("Game established")
			os.Exit(0)
		case msg := <-h.GatherPoints:
			// fmt.Printf("GHR %v/%v %v\n", msg.CurrentPlayerCount, msg.DesiredPlayerCount, msg.ConnectTo)
			err := h.JoinGatherPoint(context.TODO(), msg.ConnectTo)
			if err != nil {
				log.Err(err).Msg("Join gather point")
			}
		case <-sigCh:
			h.Close()
			return
		}
	}
}
