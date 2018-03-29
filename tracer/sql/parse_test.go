package sql

import (
	"testing"

	"github.com/xwb1989/sqlparser"
)

func TestSelectParser_Parse(t *testing.T) {
	t.Run("select", func(t *testing.T) {
		do := func(t *testing.T, sql string) {
			tree, err := sqlparser.Parse(sql)
			if err != nil {
				t.Error(err)
			}

			sel := SelectParser{Stmt: tree.(*sqlparser.Select)}
			err = sel.Parse()
			if err != nil {
				t.Error(err)
			}
		}
		t.Run("simple", func(t *testing.T) {
			do(t, "SELECT * FROM calls")
		})
		t.Run("where", func(t *testing.T) {
			do(t, "SELECT * FROM calls WHERE 1")
		})
		t.Run("where-func", func(t *testing.T) {
			do(t, "SELECT * FROM calls WHERE NOW()")
		})
		t.Run("complexity", func(t *testing.T) {
			do(t, "SELECT * FROM calls WHERE start_time > SUBTIME(NOW(), '10:0:0')")
		})
		t.Run("qualifier", func(t *testing.T) {
			do(t, "SELECT calls.id, calls.* FROM calls")
		})
		t.Run("implicit-join", func(t *testing.T) {
			do(t, "SELECT calls.*, frames.* FROM frames")
		})
	})
}
