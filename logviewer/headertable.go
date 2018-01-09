package logviewer

import (
	"github.com/marcusolsson/tui-go"
)

type headerTable struct {
	*tui.Table
	Headers []tui.Widget
	rows    int
}

func newHeaderTable(headers ...tui.Widget) *headerTable {
	return &headerTable{
		Table:   tui.NewTable(0, 0),
		Headers: headers,
	}
}
func (t *headerTable) Select(i int) {
	if t.rows <= 1 {
		t.Table.Select(-1)
	} else if i < 1 {
		t.Table.Select(1)
	} else {
		t.Table.Select(i)
	}
}
func (t *headerTable) RemoveRow(index int) {
	if index == 0 {
		// disallow remove header line.
		return
	}
	t.Table.RemoveRow(index)
	t.rows--
	if t.rows <= 1 {
		// unselect header line
		t.Table.Select(-1)
	}
}
func (t *headerTable) RemoveRows() {
	t.Table.RemoveRows()
	t.rows = 0
	t.AppendRow(t.Headers...)
}
func (t *headerTable) AppendRow(row ...tui.Widget) {
	t.Table.AppendRow(row...)
}
