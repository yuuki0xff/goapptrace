package sql

import (
	"fmt"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

func panicSqlCast(from, to string) {
	panic(errSqlCast(from, to))
}

func errSqlCast(from, to string) error {
	return fmt.Errorf("cast error: %s to %s", from, to)
}

// このシステム内で扱う全てのデータ型
type SqlAny interface {
	Bool() bool
	BigInt() int64
	String() string
	Datetime() types.Time

	// 定数として扱えるならtrueを返す。
	Const() bool
	// データ型を文字列として返す。
	Type() string
	// row には *types.FuncLog のような特定のデータ型のポインタを格納する。
	// 処理対象の行を変更したい場合は、ここで渡したポインタの先を書き換える。
	// Const() がtrueを返すなら、これを設定する必要ない。
	WithRow(row SqlRow)
}

type SqlBool bool

func (b SqlBool) Bool() bool           { return bool(b) }
func (b SqlBool) BigInt() int64        { panic(errSqlCast("bool", "bigint")) }
func (b SqlBool) String() string       { panic(errSqlCast("bool", "string")) }
func (b SqlBool) Datetime() types.Time { panic(errSqlCast("bool", "datetime")) }
func (b SqlBool) Const() bool          { return true }
func (b SqlBool) Type() string         { return "bool" }
func (b SqlBool) WithRow(row SqlRow)   {}

type SqlBigInt int64

func (b SqlBigInt) Bool() bool           { return int64(b) != 0 }
func (b SqlBigInt) BigInt() int64        { return int64(b) }
func (b SqlBigInt) String() string       { panic(errSqlCast("bigint", "string")) }
func (b SqlBigInt) Datetime() types.Time { panic(errSqlCast("bigint", "datetime")) }
func (b SqlBigInt) Const() bool          { return true }
func (b SqlBigInt) Type() string         { return "bigint" }
func (b SqlBigInt) WithRow(row SqlRow)   {}

type SqlString string

func (s SqlString) Bool() bool           { panic(errSqlCast("string", "bool")) }
func (s SqlString) BigInt() int64        { panic(errSqlCast("string", "bigint")) }
func (s SqlString) String() string       { return string(s) }
func (s SqlString) Datetime() types.Time { panic(errSqlCast("string", "datetime")) }
func (s SqlString) Const() bool          { return true }
func (s SqlString) Type() string         { return "string" }
func (s SqlString) WithRow(row SqlRow)   {}

type SqlDatetime types.Time

func (d SqlDatetime) Bool() bool           { panic(errSqlCast("datetime", "bool")) }
func (d SqlDatetime) BigInt() int64        { panic(errSqlCast("datetime", "bigint")) }
func (d SqlDatetime) String() string       { panic(errSqlCast("datetime", "string")) }
func (d SqlDatetime) Datetime() types.Time { return types.Time(d) }
func (d SqlDatetime) Const() bool          { return true }
func (d SqlDatetime) Type() string         { return "datetime" }
func (d SqlDatetime) WithRow(row SqlRow)   {}

// テーブルの1つのフィールドを表す。
// これの値を取得するときは、先にWithRow()で処理対象の行を指定すること。
type SqlField struct {
	Table, Col string
	getter     SqlFieldGetter
}

func (f *SqlField) Bool() bool           { return f.getter().Bool() }
func (f *SqlField) BigInt() int64        { return f.getter().BigInt() }
func (f *SqlField) String() string       { return f.getter().String() }
func (f *SqlField) Datetime() types.Time { return f.getter().Datetime() }
func (f *SqlField) Const() bool          { return false }
func (f *SqlField) Type() string         { return f.getter().Type() }
func (f *SqlField) WithRow(row SqlRow) {
	f.getter = row.Field(f.Table, f.Col)
}
