package sql

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

const (
	CsvFormat OutputFormat = iota
)

// SqlFieldGetter は SqlAny.WithRow() で指定した行の特定のフィールドを返す。
// 対象の行やoffsetが変更になった場合、既存の SqlFieldGetter も変更後のフィールドを返す。
type SqlFieldGetter func() SqlAny
type SqlFieldGetters []SqlFieldGetter

// SqlFieldPrinter は SqlAny.WithRow() で指定した行を特定の文字列フォーマットに変換する。
// 引数で指定したバッファに変換後の文字列を書き込み、書き込んだバイト数を戻り地として返す。
// バッファサイズが足りない場合、この関数はpanicする。
// SqlFieldGetter 同様に、処理対象の行が変更になると既存の SqlFieldPrinter も変更後の文字列を返す。
type SqlFieldPrinter func(buf []byte) int64

type OutputFormat int

func (g SqlFieldGetter) Printer(format OutputFormat) SqlFieldPrinter {
	var conv SqlFieldPrinter
	strFastcopy := func(buf []byte, sp *string) int64 {
		sh := (*reflect.StringHeader)(unsafe.Pointer(sp)) // nolint: gas
		bh := reflect.SliceHeader{
			Data: sh.Data,
			Len:  sh.Len,
			Cap:  sh.Len,
		}
		data := *(*[]byte)(unsafe.Pointer(&bh)) // nolint: gas
		return int64(copy(buf, data))
	}
	return func(buf []byte) int64 {
		if conv == nil {
			coltype := g().Type()
			switch coltype {
			case BoolType:
				conv = func(buf []byte) int64 {
					if g().Bool() {
						// true
						buf[0] = 't'
						buf[1] = 'r'
						buf[2] = 'u'
						buf[3] = 'e'
						return 4
					} else {
						// false
						buf[0] = 'f'
						buf[1] = 'a'
						buf[2] = 'l'
						buf[3] = 's'
						buf[4] = 'e'
						return 5
					}
				}
			case BigIntType:
				conv = func(buf []byte) int64 {
					val := g().BigInt()
					s := strconv.FormatInt(val, 10)
					return strFastcopy(buf, &s)
				}
			case StringType:
				conv = func(buf []byte) int64 {
					s := g().String()
					return strFastcopy(buf, &s)
				}
			case DatetimeType:
				conv = func(buf []byte) int64 {
					s := g().Datetime().UnixTime().String()
					return strFastcopy(buf, &s)
				}
			default:
				panic(fmt.Errorf("%s type is not supported", coltype))
			}
		}
		return conv(buf)
	}
}
func (gs SqlFieldGetters) Printer(format OutputFormat) (p SqlFieldPrinter) {
	switch format {
	case CsvFormat:
		for i := len(gs) - 1; i >= 0; i-- {
			g := gs[i]
			var colp SqlFieldPrinter
			if p == nil {
				p = func(buf []byte) int64 {
					if colp == nil {
						colp = g.Printer(format)
					}
					return colp(buf)
				}
			} else {
				oldp := p
				p = func(buf []byte) int64 {
					if colp == nil {
						colp = g.Printer(format)
					}
					n := colp(buf)
					buf[n] = ','
					n2 := oldp(buf[n+1:])
					return n + 1 + n2
				}
			}
		}
		return
	default:
		panic(fmt.Errorf("OutputFormat(%d) is not supported", format))
	}
}

// SqlRow は処理対象の1つの行を表すデータ型。
type SqlRow interface {
	// 指定したフィールドを返す SqlFieldGetter を作成して返す。
	// 指定したテーブルや列が存在しない場合はpanicする。
	// 複数行を処理する場合は、パフォーマンス向上のために SqlFieldGetter を再利用するべき。
	Field(table, col string) SqlFieldGetter
	// 指定したフィールドの SqlFieldGetter を全て返す。
	Fields(tables, cols []string) SqlFieldGetters
	// 対象となる types.FuncLog.Frames のインデックスを指定する。
	SetOffset(offset int)
	// 現時点での types.FuncLog.Frames の長さを返す。
	MaxOffset() int
}

type SqlFuncLogRow struct {
	// TODO: このポインタ、またはその先のデータを書き換えることで、 SqlFieldGetter が返す値を変更できる。
	FuncLog *types.FuncLog
	Symbols *types.Symbols
	offset  int
}

func (r *SqlFuncLogRow) Field(table, col string) SqlFieldGetter {
	switch table {
	case "calls":
		switch col {
		case "id":
			return func() SqlAny { return SqlBigInt(r.FuncLog.ID) }
		case "gid":
			return func() SqlAny { return SqlBigInt(r.FuncLog.GID) }
		case "starttime":
			return func() SqlAny { return SqlDatetime(r.FuncLog.StartTime) }
		case "endtime":
			return func() SqlAny { return SqlDatetime(r.FuncLog.EndTime) }
		case "exectime":
			return func() SqlAny { return SqlBigInt(r.FuncLog.EndTime - r.FuncLog.StartTime) }
		default:
			panic(fmt.Errorf("not found %s.%s column", table, col))
		}
	case "frames":
		switch col {
		case "id":
			return func() SqlAny { return SqlBigInt(r.FuncLog.ID) }
		case "offset":
			return func() SqlAny { return SqlBigInt(r.offset) }
		case "package":
			return func() SqlAny {
				f, ok := r.Symbols.GoFunc(r.FuncLog.Frames[r.offset])
				if !ok {
					return SqlString("?")
				}
				return SqlString(f.PackagePath())
			}
		case "func":
			return func() SqlAny {
				f, ok := r.Symbols.GoFunc(r.FuncLog.Frames[r.offset])
				if !ok {
					return SqlString("?")
				}
				return SqlString(f.Name)
			}
		case "file":
			return func() SqlAny {
				return SqlString(r.Symbols.File(r.FuncLog.Frames[r.offset]))
			}
		case "line":
			return func() SqlAny {
				return SqlBigInt(r.Symbols.Line(r.FuncLog.Frames[r.offset]))
			}
		case "pc":
			return func() SqlAny {
				return SqlBigInt(r.FuncLog.Frames[r.offset])
			}
		default:
			panic(fmt.Errorf("not found %s.%s column", table, col))
		}
	default:
		panic(fmt.Errorf("invalid table: %s.%s column", table, col))
	}
}
func (r *SqlFuncLogRow) Fields(tables, cols []string) SqlFieldGetters {
	if len(tables) != len(cols) {
		panic(fmt.Errorf("mismatch length: len(tables)=%d len(cols)=%d", len(tables), len(cols)))
	}
	gs := make(SqlFieldGetters, len(cols))
	for i := range gs {
		gs[i] = r.Field(tables[i], cols[i])
	}
	return gs
}
func (r *SqlFuncLogRow) SetOffset(offset int) {
	r.offset = offset
}
func (r *SqlFuncLogRow) MaxOffset() int {
	return len(r.FuncLog.Frames)
}

type SqlGoroutineRow struct {
	types.Goroutine
}

func (r *SqlGoroutineRow) Field(table, col string) SqlFieldGetter {
	switch table {
	case "goroutines":
		switch col {
		case "gid":
			return func() SqlAny { return SqlBigInt(r.GID) }
		case "starttime":
			return func() SqlAny { return SqlDatetime(r.StartTime) }
		case "endtime":
			return func() SqlAny { return SqlDatetime(r.EndTime) }
		case "exectime":
			return func() SqlAny { return SqlBigInt(r.EndTime - r.StartTime) }
		default:
			panic(fmt.Errorf("not found %s.%s column", table, col))
		}
	default:
		panic(fmt.Errorf("invalid table: %s.%s column", table, col))
	}
}
func (r *SqlGoroutineRow) Fields(tables, cols []string) SqlFieldGetters {
	if len(tables) != len(cols) {
		panic(fmt.Errorf("mismatch length: len(tables)=%d len(cols)=%d", len(tables), len(cols)))
	}
	gs := make(SqlFieldGetters, len(cols))
	for i := range gs {
		gs[i] = r.Field(tables[i], cols[i])
	}
	return gs
}
func (r *SqlGoroutineRow) SetOffset(offset int) { panic("not supported") }
func (r *SqlGoroutineRow) MaxOffset() int       { panic("not supported") }

type SqlGoFuncRow struct {
	GoFunc *types.GoFunc
}

func (r *SqlGoFuncRow) Field(table, col string) SqlFieldGetter {
	switch table {
	case "funcs":
		switch col {
		case "name":
			return func() SqlAny {
				return SqlString(r.GoFunc.Name)
			}
		case "shortname":
			return func() SqlAny {
				return SqlString(r.GoFunc.ShortName())
			}
		case "package":
			return func() SqlAny {
				return SqlString(r.GoFunc.PackagePath())
			}
		default:
			panic(fmt.Errorf("not found %s.%s column", table, col))
		}
	default:
		panic(fmt.Errorf("invalid table: %s.%s column", table, col))
	}
}
func (r *SqlGoFuncRow) Fields(tables, cols []string) SqlFieldGetters {
	if len(tables) != len(cols) {
		panic(fmt.Errorf("mismatch length: len(tables)=%d len(cols)=%d", len(tables), len(cols)))
	}
	gs := make(SqlFieldGetters, len(cols))
	for i := range gs {
		gs[i] = r.Field(tables[i], cols[i])
	}
	return gs
}
func (r *SqlGoFuncRow) SetOffset(offset int) { panic("not supported") }
func (r *SqlGoFuncRow) MaxOffset() int       { panic("not supported") }

type SqlGoModuleRow struct {
	*types.GoModule
}

func (r *SqlGoModuleRow) Field(table, col string) SqlFieldGetter {
	switch table {
	case "modules":
		switch col {
		case "module":
			return func() SqlAny {
				return SqlString(r.Name)
			}
		default:
			panic(fmt.Errorf("not found %s.%s column", table, col))
		}
	default:
		panic(fmt.Errorf("invalid table: %s.%s column", table, col))
	}
}
func (r *SqlGoModuleRow) Fields(tables, cols []string) SqlFieldGetters {
	if len(tables) != len(cols) {
		panic(fmt.Errorf("mismatch length: len(tables)=%d len(cols)=%d", len(tables), len(cols)))
	}
	gs := make(SqlFieldGetters, len(cols))
	for i := range gs {
		gs[i] = r.Field(tables[i], cols[i])
	}
	return gs
}
func (r *SqlGoModuleRow) SetOffset(offset int) { panic("not supported") }
func (r *SqlGoModuleRow) MaxOffset() int       { panic("not supported") }
