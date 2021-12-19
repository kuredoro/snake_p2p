package main

import (
	"flag"
	snake "github.com/kuredoro/snake_p2p"
	"github.com/kuredoro/snake_p2p/engine/console"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	"os"

	//"github.com/rivo/tview"
)

func main() {
	//gatherFlag := flag.Int("gather", 0, "create gather point for N players")
	flag.Parse()

	f, _ := os.Create("ui_logs")
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: f})
	ctx := context.Background()
	h, err := snake.New(ctx)
	if err != nil {
		log.Err(err).Msg("New node")
		os.Exit(1)
	}

	log.Info().Msg("Node initialized")

	g := console.NewGatherUI(h)
	// Shortcuts to navigate the slides.
	//console.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	//	if event.Key() == tcell.KeyCtrlN {
	//		//nextSlide()
	//		return nil
	//	} else if event.Key() == tcell.KeyCtrlP {
	//		//previousSlide()
	//		return nil
	//	} else if event.Key() == tcell.KeyEscape {
	//		console.Cover()
	//		return nil
	//	}
	//	return event
	//})

	// Start the application.
	if err := g.Run(); err != nil {
		panic("Game Run finished with error")
	}
}
