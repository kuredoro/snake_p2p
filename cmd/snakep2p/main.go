package main

import (
	"flag"
	"os"

	snake "github.com/kuredoro/snake_p2p"
	"github.com/kuredoro/snake_p2p/engine/console"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
	//"github.com/rivo/tview"
)

func main() {
	// gatherFlag := flag.Int("gather", 0, "create gather point for N players")
	logNameFlag := flag.String("logname", "ui_logs.txt", "Name of log file")
	flag.Parse()

	f, _ := os.Create(*logNameFlag)
	log.Logger = log.Output(f)
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
		panic("GameUI Run finished with error")
	}
}
