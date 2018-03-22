package sql

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSqlBool_Bool(t *testing.T) {
	a := assert.New(t)
	a.True(SqlBool(true).Bool())
	a.False(SqlBool(false).Bool())
}
func TestSqlBool_BigInt(t *testing.T) {
	a := assert.New(t)
	a.Panics(func() {
		SqlBool(true).BigInt()
	})
}
func TestSqlBool_Const(t *testing.T) {
	a := assert.New(t)
	a.True(SqlBool(false).Const())
}
func TestSqlBool_Type(t *testing.T) {
	a := assert.New(t)
	a.Equal("bool", SqlBool(false).Type())
}
func TestSqlBool_WithRow(t *testing.T) {
	a := assert.New(t)
	a.NotPanics(func() {
		// SqlBool は WithRow() を無視しなければならない。
		SqlBool(false).WithRow(nil)
	})
}
