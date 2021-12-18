package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	snake "github.com/kuredoro/snake_p2p"
)

func main() {
	peerAddrFlag := flag.String("peer", "", "peer to connect to")
	gatherFlag := flag.Bool("gather", false, "should this peer announce a gather point?")
	flag.Parse()

	ctx := context.Background()
	h, err := snake.New(ctx)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	if *peerAddrFlag != "" {
		/*
			pi, err := peer.AddrInfoFromString(*peerAddrFlag)
			if err != nil {
				printErr("parse peer p2p multiaddr:", err)
				os.Exit(1)
			}

			err = h.Connect(context.Background(), *pi)
			if err != nil {
				fmt.Printf("ERR connecting to peer %v: %v\n", pi.ID.Pretty(), err)
			}
		*/
	}

	if *gatherFlag {
		h.CreateGatherPoint(time.Second)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Read from the channel and send
	for {
		select {
		case <-h.EstablishedGames:
			fmt.Printf("established a game\n")
			os.Exit(0)
		case msg := <-h.GatherPoints:
			// fmt.Printf("GHR %v/%v %v\n", msg.CurrentPlayerCount, msg.DesiredPlayerCount, msg.ConnectTo)
			err := h.JoinGatherPoint(context.TODO(), msg.ConnectTo)
			if err != nil {
				fmt.Printf("ERR join gather point: %v\n", err)
			}
		case <-sigCh:
			h.Close()
			return
		}
	}
}
