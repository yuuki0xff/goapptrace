package logviewer

import (
	"github.com/marcusolsson/tui-go"
)

// ヘッダーを指定可能なテーブル
type headerTable struct {
	*tui.Table
	Headers []tui.Widget
	// ヘッダーを除いた行数
	rows int

	onActivated func(table *tui.Table)
	onSelected  func(table *tui.Table)
}

func newHeaderTable(headers ...tui.Widget) *headerTable {
	table := tui.NewTable(0, 0)
	table.AppendRow(headers...)
	table.Select(-1)

	t := &headerTable{
		Table:       table,
		Headers:     headers,
		rows:        0,
		onActivated: discardTableEvent,
		onSelected:  discardTableEvent,
	}
	t.Table.OnItemActivated(func(table *tui.Table) {
		t.onActivated(table)
	})
	t.Table.OnSelectionChanged(func(table *tui.Table) {
		if t.rows == 0 && t.Selected() == 0 {
			t.Table.Select(-1)
			return
		} else if t.rows > 0 && t.Selected() <= 0 {
			t.Table.Select(1)
			return
		}

		t.onSelected(table)
	})
	return t
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
	t.rows++
}

func (t *headerTable) OnItemActivated(fn func(table *tui.Table)) {
	if fn == nil {
		t.onActivated = discardTableEvent
	} else {
		t.onActivated = fn
	}
}

func (t *headerTable) OnSelectionChanged(fn func(table *tui.Table)) {
	if fn == nil {
		t.onSelected = discardTableEvent
	} else {
		t.onSelected = fn
	}
}

func discardTableEvent(table *tui.Table) {}
