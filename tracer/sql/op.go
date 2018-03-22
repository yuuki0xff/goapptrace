package sql

import (
	"fmt"

	"github.com/xwb1989/sqlparser"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type AndOp struct {
	Left  SqlAny
	Right SqlAny
	// TODO: cacheを導入する
}

func (o *AndOp) Bool() bool {
	l := o.Left.Bool()
	r := o.Right.Bool()
	return l && r
}
func (o *AndOp) BigInt() int64        { panic(errSqlCast(BoolType, BigIntType)) }
func (o *AndOp) String() string       { panic(errSqlCast(BoolType, StringType)) }
func (o *AndOp) Datetime() types.Time { panic(errSqlCast(BoolType, DatetimeType)) }
func (o *AndOp) Const() bool          { return o.Left.Const() && o.Right.Const() }
func (o *AndOp) Type() string         { return BoolType }
func (o *AndOp) WithRow(row SqlRow) {
	o.Left.WithRow(row)
	o.Right.WithRow(row)
}

type OrOp struct {
	Left  SqlAny
	Right SqlAny
	// TODO: cacheを導入する
}

func (o *OrOp) Bool() bool {
	l := o.Left.Bool()
	r := o.Right.Bool()
	return l || r
}
func (o *OrOp) BigInt() int64        { panic(errSqlCast(BoolType, BigIntType)) }
func (o *OrOp) String() string       { panic(errSqlCast(BoolType, StringType)) }
func (o *OrOp) Datetime() types.Time { panic(errSqlCast(BoolType, DatetimeType)) }
func (o *OrOp) Const() bool          { return o.Left.Const() && o.Right.Const() }
func (o *OrOp) Type() string         { return BoolType }
func (o *OrOp) WithRow(row SqlRow) {
	o.Left.WithRow(row)
	o.Right.WithRow(row)
}

type NotOp struct {
	Expr SqlAny
}

func (o *NotOp) Bool() bool           { return !o.Expr.Bool() }
func (o *NotOp) BigInt() int64        { panic(errSqlCast(BoolType, BigIntType)) }
func (o *NotOp) String() string       { panic(errSqlCast(BoolType, StringType)) }
func (o *NotOp) Datetime() types.Time { panic(errSqlCast(BoolType, DatetimeType)) }
func (o *NotOp) Const() bool          { return o.Expr.Const() }
func (o *NotOp) Type() string         { return BoolType }
func (o *NotOp) WithRow(row SqlRow)   { o.Expr.WithRow(row) }

type CompOp struct {
	Operator string
	Left     SqlAny
	Right    SqlAny
	compare  func() bool
	// TODO: cacheを導入する
}

func (o *CompOp) Bool() bool {
	if o.compare == nil {
		o.compare = o.compareFn()
	}
	return o.compare()
}
func (o *CompOp) BigInt() int64        { panic(errSqlCast(BoolType, BigIntType)) }
func (o *CompOp) String() string       { panic(errSqlCast(BoolType, StringType)) }
func (o *CompOp) Datetime() types.Time { panic(errSqlCast(BoolType, DatetimeType)) }
func (o *CompOp) Const() bool          { return o.Left.Const() && o.Right.Const() }
func (o *CompOp) Type() string         { return BoolType }
func (o *CompOp) WithRow(row SqlRow) {
	o.Left.WithRow(row)
	o.Right.WithRow(row)
}
func (o *CompOp) compareFn() func() bool {
	t := o.Left.Type()
	t2 := o.Right.Type()
	if t != t2 {
		panic(fmt.Errorf("mismatch type: %s %s %s", t, o.Operator, t2))
	}
	switch t {
	case BoolType:
		switch o.Operator {
		case sqlparser.EqualStr:
			return func() bool {
				return o.Left.Bool() == o.Right.Bool()
			}
		case sqlparser.NotEqualStr:
			return func() bool {
				return o.Left.Bool() != o.Right.Bool()
			}
		case sqlparser.NullSafeEqualStr:
			return func() bool {
				return o.Left.Bool() == o.Right.Bool()
			}
		default:
			panic(fmt.Errorf("not supported operator: %s %s %s", t, o.Operator, t2))
		}
	case BigIntType:
		switch o.Operator {
		case sqlparser.EqualStr:
			return func() bool { return o.Left.BigInt() == o.Right.BigInt() }
		case sqlparser.LessThanStr:
			return func() bool { return o.Left.BigInt() < o.Right.BigInt() }
		case sqlparser.GreaterThanStr:
			return func() bool { return o.Left.BigInt() > o.Right.BigInt() }
		case sqlparser.LessEqualStr:
			return func() bool { return o.Left.BigInt() <= o.Right.BigInt() }
		case sqlparser.GreaterEqualStr:
			return func() bool { return o.Left.BigInt() >= o.Right.BigInt() }
		case sqlparser.NotEqualStr:
			return func() bool { return o.Left.BigInt() != o.Right.BigInt() }
		case sqlparser.NullSafeEqualStr:
			return func() bool { return o.Left.BigInt() == o.Right.BigInt() }
		default:
			panic(fmt.Errorf("not supported operator: %s %s %s", t, o.Operator, t2))
		}
	case StringType:
		switch o.Operator {
		case sqlparser.EqualStr:
			return func() bool { return o.Left.String() == o.Right.String() }
		case sqlparser.NotEqualStr:
			return func() bool { return o.Left.String() != o.Right.String() }
		case sqlparser.NullSafeEqualStr:
			return func() bool { return o.Left.String() == o.Right.String() }
		case sqlparser.LikeStr:
			panic("todo") // TODO
		case sqlparser.NotLikeStr:
			panic("todo") // TODO
		case sqlparser.RegexpStr:
			panic("todo") // TODO
		case sqlparser.NotRegexpStr:
			panic("todo") // TODO
		default:
			panic(fmt.Errorf("not supported operator: %s %s %s", t, o.Operator, t2))
		}
	default:
		panic(fmt.Errorf("bug: type=%s", t))
	}
}

type RangeOp struct {
	Left SqlAny
	From SqlAny
	To   SqlAny
	// TODO: cacheを導入する
}

func (r *RangeOp) Bool() bool {
	val := r.Left.BigInt()
	from := r.From.BigInt()
	to := r.To.BigInt()
	return from <= val && val <= to
}
func (r *RangeOp) BigInt() int64        { panic(errSqlCast(BoolType, BigIntType)) }
func (r *RangeOp) String() string       { panic(errSqlCast(BoolType, StringType)) }
func (r *RangeOp) Datetime() types.Time { panic(errSqlCast(BoolType, DatetimeType)) }
func (r *RangeOp) Const() bool          { return r.Left.Const() && r.From.Const() && r.To.Const() }
func (r *RangeOp) Type() string         { return BoolType }
func (r *RangeOp) WithRow(row SqlRow) {
	r.Left.WithRow(row)
	r.From.WithRow(row)
	r.To.WithRow(row)
}
