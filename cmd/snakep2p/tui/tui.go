package tui

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	snake "github.com/kuredoro/snake_p2p"
	"github.com/kuredoro/snake_p2p/protocol/gather"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

type GatherUI struct {
	h             *snake.Node
	app           *tview.Application
	flex          *tview.Flex
	myGatherPoint *tview.TextView
	gameList      *tview.Table
	createBtn     *tview.Button
	newGame       *tview.InputField
	maxPlayers    int
	gatherPoints  map[string]*gather.GatherPointMessage
}

func addRow(table *tview.Table, msg *gather.GatherPointMessage, row int, color tcell.Color) {
	ID := msg.ConnectTo.ID.Pretty()
	tableCell := tview.NewTableCell(ID).
		SetTextColor(color).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(row, 0, tableCell)
	maxPlayers := strconv.Itoa(int(msg.DesiredPlayerCount))
	tableCell = tview.NewTableCell(maxPlayers).
		SetTextColor(color).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(row, 1, tableCell)
	tableCell = tview.NewTableCell("").
		SetTextColor(color).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(row, 2, tableCell)
}

func NewGatherUI(h *snake.Node) *GatherUI {
	g := &GatherUI{}
	g.h = h
	g.app = tview.NewApplication()
	g.gatherPoints = make(map[string]*gather.GatherPointMessage)
	g.myGatherPoint = tview.NewTextView().
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() { g.app.Draw() })
	g.myGatherPoint.SetBorder(true).SetTitle("My Gather Point")
	fmt.Fprintf(g.myGatherPoint, "No gather point created.")

	table := tview.NewTable().
		SetFixed(1, 0).
		SetSelectable(true, false)
	tableCell := tview.NewTableCell("ID").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(1, 0, tableCell)
	tableCell = tview.NewTableCell("Players needed").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(1, 1, tableCell)
	tableCell = tview.NewTableCell("Joined").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(1, 2, tableCell)
	g.gameList = table
	g.gameList.SetSelectedFunc(func(row, column int) {
		if row == 1 {
			return
		}
		ID := g.gameList.GetCell(row, 0).Text
		msg := g.gatherPoints[ID]
		ctx := context.Background()
		err := g.h.JoinGatherPoint(ctx, msg.ConnectTo)
		if err != nil {
			log.Err(err).Msg("Join gather point")
		}
		// cell := table.GetCell(row, 2)
		// cell.Text = "✔️"
		g.gameList.GetCell(row, 2).SetText("〇")
	})

	g.newGame = tview.NewInputField().
		SetLabel("Enter the maximum number of players ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetAcceptanceFunc(tview.InputFieldInteger)

	g.newGame.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			// we don't want to do anything if they just tabbed away
			return
		}
		g.maxPlayers, _ = strconv.Atoi(g.newGame.GetText())
		err := g.h.CreateGatherPoint(g.maxPlayers, time.Second)
		if err != nil {
			log.Err(err).Msg("New gather point")
		}
		g.myGatherPoint.Clear()
		fmt.Fprintf(g.myGatherPoint, "Max # of players: %d", g.maxPlayers)
		g.flex.RemoveItem(g.newGame)
	})

	g.createBtn = tview.NewButton("Create gather point").SetSelectedFunc(func() {
		g.flex.RemoveItem(g.createBtn)
		g.flex.AddItem(g.newGame, 0, 1, false)
	})

	g.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(g.myGatherPoint, 3, 1, false).
		AddItem(g.gameList, 0, 3, false).
		AddItem(g.createBtn, 2, 1, false)

	g.app.SetRoot(g.flex, true).EnableMouse(true)
	return g
}

func (g *GatherUI) eventLoop() {
	sigCh := make(chan os.Signal, 1)
	for {
		select {
		case info := <-g.h.EstablishedGames:
			log.Info().
				Str("facilitator", info.Facilitator.Pretty()).
				Int("peer_count", info.Game.PeerCount()).
				Msg("GameUI established")
			gi := info.Game
			game := NewGame(gi)
			g.app.Suspend(func() {
				seed := gi.Run()
				game.RunGame(seed)
				gi.Close()
			})
			//for i := 0; i < 3; i++ {
			//	err := gi.SendMove(core.Up)
			//	if err != nil {
			//		log.Err(err).Msg("Test move")
			//	}
			//
			//	log.Info().Msg("Sent move")
			//
			//	move := <-gi.IncommingMoves()
			//	for peer, dir := range move.Moves {
			//		log.Info().
			//			Str("peer", peer.Pretty()).
			//			Int("dir", int(dir)).
			//			Msg("Player moved")
			//	}
			//}
		case msg := <-g.h.GatherPoints:
			if _, exists := g.gatherPoints[msg.ConnectTo.ID.Pretty()]; exists {
				continue
			}

			log.Info().
				Str("facilitator", msg.ConnectTo.ID.Pretty()).
				Uint("desired_player_count", msg.DesiredPlayerCount).
				Msg("Found new gather point")

			g.gatherPoints[msg.ConnectTo.ID.Pretty()] = msg
			// Add cell to gather points table
			addRow(g.gameList, msg, len(g.gatherPoints)+1, tcell.ColorWhite)
			g.app.Draw()
		case <-sigCh:
			g.h.Close()
			return
		}
	}
}

func (g *GatherUI) Run() error {
	go g.eventLoop()
	return g.app.Run()
}
