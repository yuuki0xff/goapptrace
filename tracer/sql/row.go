package sql

import (
	"fmt"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// SqlFieldGetter は SqlAny.WithRow() で指定した行の特定のフィールドを返す。
// 対象の行やoffsetが変更になった場合、既存の SqlFieldGetter も変更後のフィールドを返す。
type SqlFieldGetter func() SqlAny

// SqlRow は処理対象の1つの行を表すデータ型。
type SqlRow interface {
	// 指定したフィールドを返す SqlFieldGetter を作成して返す。
	// 指定したテーブルや列が存在しない場合はpanicする。
	// 複数行を処理する場合は、パフォーマンス向上のために SqlFieldGetter を再利用するべき。
	Field(table, col string) SqlFieldGetter
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
func (r *SqlFuncLogRow) SetOffset(offset int) {
	r.offset = offset
}
func (r *SqlFuncLogRow) MaxOffset() int {
	return len(r.FuncLog.Frames)
}

type SqlGoroutineRow types.Goroutine

func (r *SqlGoroutineRow) Field(table, col string, offset int) SqlFieldGetter {
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
func (r *SqlGoroutineRow) SetOffset(offset int) { panic("not supported") }
func (r *SqlGoroutineRow) MaxOffset() int       { panic("not supported") }

type SqlGoFuncRow struct {
	GoFunc *types.GoFunc
}

func (r *SqlGoFuncRow) Field(table, col string, offset int) SqlFieldGetter {
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
func (r *SqlGoFuncRow) SetOffset(offset int) { panic("not supported") }
func (r *SqlGoFuncRow) MaxOffset() int       { panic("not supported") }

type SqlGoModuleRow types.GoModule

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
func (r *SqlGoModuleRow) SetOffset(offset int) { panic("not supported") }
func (r *SqlGoModuleRow) MaxOffset() int       { panic("not supported") }
