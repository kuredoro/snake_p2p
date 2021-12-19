package console

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strconv"
)

type GatherUI struct {
	app *tview.Application
	flex *tview.Flex
	myGatherPoint *tview.TextView
	gameList *tview.Table
	createBtn *tview.Button
	newGame *tview.InputField
	maxPlayers int
	joinGames int
}

func checkNewGameField(textToCheck string, lastChar rune) bool {
	_, err := strconv.Atoi(textToCheck)
	if err != nil {
		return false
	}
	return true
}

func GatherUIInit() *GatherUI {
	g := &GatherUI{}
	g.app = tview.NewApplication()
	g.myGatherPoint = tview.NewTextView().
						SetRegions(true).
						SetDynamicColors(true).
						SetWordWrap(true).
						SetChangedFunc(func() { g.app.Draw() })
	g.myGatherPoint.SetBorder(true).SetTitle("My Gather Point")
	fmt.Fprintf(g.myGatherPoint, "No gather point created.")

	table := tview.NewTable().
		SetFixed(1, 1).
		SetSelectable(true, false)
	tableCell := tview.NewTableCell("ID").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(1, 1, tableCell)
	tableCell = tview.NewTableCell("Players needed").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(1, 2, tableCell)
	tableCell = tview.NewTableCell("Signed").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	table.SetCell(1, 3, tableCell)
	g.gameList = table

	g.newGame = tview.NewInputField().
					SetLabel("Enter the maximum number of players ").
					SetFieldWidth(0).
					SetFieldBackgroundColor(tcell.ColorBlack)

	g.newGame.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			// we don't want to do anything if they just tabbed away
			return
		}
		line := g.newGame.GetText()
		if _, err := strconv.Atoi(line); err != nil {
			//print("It is not number", err)
			// ignore not numbers
			return
		}
		g.maxPlayers, _ = strconv.Atoi(line)
		g.myGatherPoint.Clear()
		fmt.Fprintf(g.myGatherPoint, "Max # of players: %d", g.maxPlayers)
		g.flex.RemoveItem(g.newGame)
	})

	g.createBtn = tview.NewButton("Create Game").SetSelectedFunc(func() {
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

func (gatherUI *GatherUI)Run() error {
	return gatherUI.app.Run()
}