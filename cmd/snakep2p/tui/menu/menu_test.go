package menu_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/kuredoro/snake_p2p/cmd/snakep2p/tui/menu"
)

var tagPattern = regexp.MustCompile(`\[([a-zA-Z]+|#[0-9a-zA-Z]{6}|\-)?(:([a-zA-Z]+|#[0-9a-zA-Z]{6}|\-)?(:([01]+|[bdilrsu]+|\-)?)?)?\]`)

type Tag tcell.Style

func NewTag(str string) Tag {
	style := tcell.StyleDefault

	parts := tagPattern.FindStringSubmatch(str)

	if len(parts) >= 2 && parts[1] != "" && parts[1] != "-" {
		style = style.Foreground(tcell.GetColor(parts[1]))
	}

	if len(parts) >= 4 && parts[3] != "" && parts[3] != "-" {
		style = style.Background(tcell.GetColor(parts[3]))
	}

	if len(parts) >= 6 && parts[5] != "" && parts[5] != "-" {
		mask, err := strconv.ParseInt(parts[5], 2, 64)
		if err != nil {
			panic(fmt.Errorf("attempt to create a tag from a malformed string %q: attributes: %v", str, err))
		}

		style = style.Attributes(tcell.AttrMask(mask))
	}

	return Tag(style)
}

func (t Tag) String() string {
	fg, bg, attr := tcell.Style(t).Decompose()

	return fmt.Sprintf("[#%06x:#%06x:%b]", fg.Hex(), bg.Hex(), attr)
}

func (t Tag) Style() tcell.Style {
	return tcell.Style(t)
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func AssertSimulationScreen(t *testing.T, got tcell.SimulationScreen, want []string) {
	t.Helper()

	gotCells, w, h := got.GetContents()
	wantCells, wantWidth, wantHeight := SimCellsFromStrings(want)

	if w != wantWidth || h != wantHeight {
		t.Fatalf("got simulation screen of size %dx%d, want %dx%d", w, h, wantWidth, wantHeight)
		return
	}

	if len(gotCells) != len(wantCells) {
		t.Fatalf("got simulation screen that contains %d cells, want %d, even though the "+
			"reported dimensions (%dx%d) coincide", len(gotCells), len(wantCells), w, h)
	}

	for i := range gotCells {
		if len(gotCells[i].Runes) == 0 && len(wantCells[i].Runes) == 1 && wantCells[i].Runes[0] == ' ' {
			continue
		}

		if !runesEqual(gotCells[i].Runes, wantCells[i].Runes) {
			t.Errorf("at %dx%d got simcell with contents %v, want %v", i%w+1, i/w+1,
				gotCells[i].Runes, wantCells[i].Runes)
		}
	}
}

func SimCellsFromStrings(rows []string) ([]tcell.SimCell, int, int) {
	if len(rows) == 0 {
		return nil, 0, 0
	}

	width := len(rows[0])
	for i := range rows {
		if len(rows[i]) != width {
			panic(fmt.Sprintf("inconsistent simulation screen row dimensions: "+
				"row #1 being %d columns wide, while row #%d being %d",
				width, i+1, len(rows[i])))
		}
	}

	cells := make([]tcell.SimCell, len(rows)*width)
	for y := range rows {
		for x, r := range rows[y] {
			cells[width*y+x].Runes = []rune{r}
		}
	}

	return cells, width, len(rows)
}

func TestMenu(t *testing.T) {
	t.Run("empty menu draws nothing", func(t *testing.T) {
		s := tcell.NewSimulationScreen("UTF-8")
		s.SetSize(4, 4)

		m := menu.New(nil)

		m.Draw(s)

		want := []string{
			"    ",
			"    ",
			"    ",
			"    ",
		}

		AssertSimulationScreen(t, s, want)
	})
}
