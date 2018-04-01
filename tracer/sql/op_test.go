package sql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type SqlMock struct {
	BoolValue    bool
	BoolIsCalled bool

	BigIntValue    int64
	BigIntIsCalled bool

	StringValue    string
	StringIsCalled bool

	DatetimeValue    types.Time
	DatetimeIsCalled bool

	IsConst       bool
	ConstIsCalled bool

	TypeStr      string
	TypeIsCalled bool

	Row             SqlRow
	WithRowIsCalled bool
}

func (m *SqlMock) Bool() bool {
	m.BoolIsCalled = true
	return m.BoolValue
}
func (m *SqlMock) BigInt() int64 {
	m.BigIntIsCalled = true
	return m.BigIntValue
}
func (m *SqlMock) String() string {
	m.StringIsCalled = true
	return m.StringValue
}
func (m *SqlMock) Datetime() types.Time {
	m.DatetimeIsCalled = true
	return m.DatetimeValue
}
func (m *SqlMock) Const() bool {
	m.ConstIsCalled = true
	return m.IsConst
}
func (m *SqlMock) Type() string {
	m.TypeIsCalled = true
	return m.TypeStr
}
func (m *SqlMock) WithRow(row SqlRow) {
	m.WithRowIsCalled = true
	m.Row = row
}

func TestAndOp_Bool(t *testing.T) {
	a := assert.New(t)
	and := AndOp{
		Left:  &SqlMock{BoolValue: true},
		Right: &SqlMock{BoolValue: true},
	}
	a.True(and.Bool())

	and = AndOp{
		Left:  &SqlMock{BoolValue: false},
		Right: &SqlMock{BoolValue: true},
	}
	a.False(and.Bool())
}
func TestAndOp_Const(t *testing.T) {
	a := assert.New(t)
	and := AndOp{
		Left:  &SqlMock{IsConst: true},
		Right: &SqlMock{IsConst: true},
	}
	a.True(and.Const())

	and = AndOp{
		Left:  &SqlMock{IsConst: false},
		Right: &SqlMock{IsConst: true},
	}
	a.False(and.Const())
}
func TestAndOp_Type(t *testing.T) {
	a := assert.New(t)
	and := AndOp{}
	a.Equal(BoolType, and.Type())
}
func TestAndOp_WithRow(t *testing.T) {
	a := assert.New(t)
	and := AndOp{
		Left:  &SqlMock{},
		Right: &SqlMock{},
	}
	a.NotPanics(func() {
		and.WithRow(nil)
	})
	a.True(and.Left.(*SqlMock).WithRowIsCalled)
	a.True(and.Right.(*SqlMock).WithRowIsCalled)
}

func TestOrOp_Bool(t *testing.T) {
	a := assert.New(t)
	or := OrOp{
		Left:  &SqlMock{BoolValue: false},
		Right: &SqlMock{BoolValue: true},
	}
	a.True(or.Bool())
	or = OrOp{
		Left:  &SqlMock{BoolValue: false},
		Right: &SqlMock{BoolValue: false},
	}
	a.False(or.Bool())
}

func TestNotOp_Bool(t *testing.T) {
	a := assert.New(t)
	not := NotOp{
		Expr: &SqlMock{BoolValue: true},
	}
	a.False(not.Bool())
	not = NotOp{
		Expr: &SqlMock{BoolValue: false},
	}
	a.True(not.Bool())
}

func TestCompOp_Bool(t *testing.T) {
	t.Run("mismatch-type", func(t *testing.T) {
		a := assert.New(t)
		comp := CompOp{
			Operator: "=",
			Left:     &SqlMock{TypeStr: BoolType},
			Right:    &SqlMock{TypeStr: StringType},
		}
		a.Panics(func() {
			comp.Bool()
		})
	})
	t.Run(BoolType, func(t *testing.T) {
		a := assert.New(t)
		comp := CompOp{
			Operator: "=",
			Left:     SqlBool(true),
			Right:    SqlBool(true),
		}
		a.True(comp.Bool())

		comp = CompOp{
			Operator: ">",
			Left:     SqlBool(true),
			Right:    SqlBool(true),
		}
		a.Panics(func() {
			comp.Bool()
		})
	})
	t.Run(BigIntType, func(t *testing.T) {
		a := assert.New(t)
		comp := CompOp{
			Operator: ">",
			Left:     SqlBigInt(20),
			Right:    SqlBigInt(10),
		}
		a.True(comp.Bool())

		comp = CompOp{
			Operator: "like",
			Left:     SqlBigInt(20),
			Right:    SqlBigInt(10),
		}
		a.Panics(func() {
			comp.Bool()
		})
	})
	t.Run(StringType, func(t *testing.T) {
		a := assert.New(t)
		comp := CompOp{
			Operator: "=",
			Left:     SqlString("ok"),
			Right:    SqlString("ok"),
		}
		a.True(comp.Bool())
		// TODO
	})
}
