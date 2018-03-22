package sql

import (
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

var funcs = []SqlFunc{
	{
		Name:  "FRAME",
		Table: "calls",
		Parse: func(args ...SqlAny) SqlAny {
			if len(args) < 1 {
				panic("missing args")
			}
			if len(args) > 1 {
				panic("too many args")
			}
			return &SqlFuncFrame{
				Expr: args[0],
			}
		},
	}, {
		Name:  "CALL",
		Table: "calls",
		Parse: func(args ...SqlAny) SqlAny {
			if len(args) < 1 {
				panic("missing args")
			}
			if len(args) > 1 {
				panic("too many args")
			}
			return &SqlFuncCall{
				Expr: args[0],
			}
		},
	}, {
		Name: "NOW",
		Parse: func(args ...SqlAny) SqlAny {
			if len(args) > 0 {
				panic("too many args")
			}
			return &SqlFuncNow{}
		},
	}, {
		Name: "ADDTIME",
		Parse: func(args ...SqlAny) SqlAny {
			if len(args) != 2 {
				panic("invalid args")
			}
			return &SqlFuncAddTime{}
		},
	}, {
		Name: "SUBTIME",
		Parse: func(args ...SqlAny) SqlAny {
			if len(args) != 2 {
				panic("invalid args")
			}
			return &SqlFuncSubTime{}
		},
	},
}

type SqlFunc struct {
	Name string
	// 関数の引数で、テーブル名が省略された場合に補完するテーブル名。
	// 空の場合は、関数呼び出し元の設定が適用される。
	Table string
	Parse func(args ...SqlAny) SqlAny
}

type SqlFuncFrame struct {
	Expr SqlAny
	row  *SqlFuncLogRow
	// TODO: cacheを導入する
}

func (d SqlFuncFrame) Bool() bool {
	max := d.row.MaxOffset()
	for i := 0; i < max; i++ {
		d.row.SetOffset(i)
		if d.Expr.Bool() {
			return true
		}
	}
	return false
}
func (d SqlFuncFrame) BigInt() int64        { panic(errSqlCast(BoolType, BigIntType)) }
func (d SqlFuncFrame) String() string       { panic(errSqlCast(BoolType, StringType)) }
func (d SqlFuncFrame) Datetime() types.Time { panic(errSqlCast(BoolType, DatetimeType)) }
func (d SqlFuncFrame) Const() bool          { return false }
func (d SqlFuncFrame) Type() string         { return BoolType }
func (d SqlFuncFrame) WithRow(row SqlRow) {
	d.row = row.(*SqlFuncLogRow)
	d.Expr.WithRow(row)
}

type SqlFuncCall struct {
	Expr SqlAny
	row  *SqlFuncLogRow
	// TODO: cacheを導入する
}

func (d SqlFuncCall) Bool() bool           { return d.Expr.Bool() }
func (d SqlFuncCall) BigInt() int64        { panic(errSqlCast(BoolType, BigIntType)) }
func (d SqlFuncCall) String() string       { panic(errSqlCast(BoolType, StringType)) }
func (d SqlFuncCall) Datetime() types.Time { panic(errSqlCast(BoolType, DatetimeType)) }
func (d SqlFuncCall) Const() bool          { return false }
func (d SqlFuncCall) Type() string         { return BoolType }
func (d SqlFuncCall) WithRow(row SqlRow) {
	d.row = row.(*SqlFuncLogRow)
	d.Expr.WithRow(row)
}

type SqlFuncNow struct {
	// TODO: cacheを導入する
}

func (d SqlFuncNow) Bool() bool           { panic(errSqlCast(DatetimeType, BoolType)) }
func (d SqlFuncNow) BigInt() int64        { panic(errSqlCast(DatetimeType, BigIntType)) }
func (d SqlFuncNow) String() string       { panic(errSqlCast(DatetimeType, StringType)) }
func (d SqlFuncNow) Datetime() types.Time { return types.NewTime(time.Now()) }
func (d SqlFuncNow) Const() bool          { return false }
func (d SqlFuncNow) Type() string         { return DatetimeType }
func (d SqlFuncNow) WithRow(row SqlRow)   {}

type SqlFuncAddTime struct {
	Base SqlAny
	Diff SqlAny
	// TODO: cacheを導入する
}

func (d SqlFuncAddTime) Bool() bool           { panic(errSqlCast(DatetimeType, BoolType)) }
func (d SqlFuncAddTime) BigInt() int64        { panic(errSqlCast(DatetimeType, BigIntType)) }
func (d SqlFuncAddTime) String() string       { panic(errSqlCast(DatetimeType, StringType)) }
func (d SqlFuncAddTime) Datetime() types.Time { return d.Base.Datetime() + d.Diff.Datetime() }
func (d SqlFuncAddTime) Const() bool          { return false }
func (d SqlFuncAddTime) Type() string         { return DatetimeType }
func (d SqlFuncAddTime) WithRow(row SqlRow)   {}

type SqlFuncSubTime struct {
	Base SqlAny
	Diff SqlAny
	// TODO: cacheを導入する
}

func (d SqlFuncSubTime) Bool() bool           { panic(errSqlCast(DatetimeType, BoolType)) }
func (d SqlFuncSubTime) BigInt() int64        { panic(errSqlCast(DatetimeType, BigIntType)) }
func (d SqlFuncSubTime) String() string       { panic(errSqlCast(DatetimeType, StringType)) }
func (d SqlFuncSubTime) Datetime() types.Time { return d.Base.Datetime() - d.Diff.Datetime() }
func (d SqlFuncSubTime) Const() bool          { return false }
func (d SqlFuncSubTime) Type() string         { return DatetimeType }
func (d SqlFuncSubTime) WithRow(row SqlRow)   {}
