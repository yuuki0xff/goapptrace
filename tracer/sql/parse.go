package sql

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/xwb1989/sqlparser"
	"github.com/yuuki0xff/goapptrace/tracer/util"
)

var (
	ErrDistinct          = errors.New("DISTINCT is not supported")
	ErrNotFoundTable     = errors.New("not found table")
	ErrJoin              = errors.New("JOIN is not supported")
	ErrTableAlias        = errors.New("table alias is not supported")
	ErrTableQualifier    = errors.New("table qualifier is not supported")
	ErrDBQualifier       = errors.New("database name qualifier is not supported")
	ErrSubquery          = errors.New("subquery is not supported")
	ErrStar              = errors.New("\"*\" and other columns are exclusive")
	ErrColumnAlias       = errors.New("column alias is not supported")
	ErrColumnQualifier   = errors.New("column qualifier is not supported")
	ErrColumnList        = errors.New("column list MUST NOT contain anything other than field names")
	ErrUnsupportedStmt   = errors.New("this statement is not supported")
	ErrGroupBy           = errors.New("GROUP BY is not supported")
	ErrHaving            = errors.New("HAVING is not supported")
	ErrOrderBy           = errors.New("ORDER BY is not supported")
	ErrLimit             = errors.New("LIMIT is not supported")
	ErrFunctionQualifier = errors.New("function qualifier is not supported")

	tables = []Table{
		{
			Name: "calls",
			Fields: []string{
				"id", "gid", "starttime", "endtime", "exectime",
			},
		}, {
			Name: "frames",
			Fields: []string{
				"id", "offset", "package", "func", "file", "line", "pc",
			},
			ImplictJoin: "calls",
		}, {
			Name: "goroutines",
			Fields: []string{
				"gid", "starttime", "endtime", "exectime",
			},
		}, {
			Name: "funcs",
			Fields: []string{
				"name", "shortname", "package", "path",
			},
		}, {
			Name: "modules",
			Fields: []string{
				"module",
			},
		},
	}
)

type Table struct {
	Name   string
	Fields []string
	// a table name that can be implicitly joined.
	ImplictJoin string
}

func (t Table) HasField(name string) bool {
	for i := range t.Fields {
		if name == t.Fields[i] {
			return true
		}
	}
	return false
}

// findTableByName finds a table by table name.
func findTableByName(name string) (Table, bool) {
	for i := range tables {
		if name == tables[i].Name {
			return tables[i], true
		}
	}
	return Table{}, false
}

type Field struct {
	// table name
	Table string
	// field name
	Name string
	// display name
	AliasName string
}

func (f Field) LongName() string {
	return f.Table + "." + f.Name
}
func (f Field) String() string {
	if f.AliasName != "" {
		return f.AliasName
	}
	return f.LongName()
}

type SelectParser struct {
	Stmt *sqlparser.Select

	table Table
	// フィールド名のリスト
	fields []Field

	where  SqlAny
	offset int64
	rows   int64
}

// parseSelect parses a "SELECT" statement.
func (s *SelectParser) Parse() error {
	if s.Stmt.Distinct != "" {
		return ErrDistinct
	}
	tname, err := s.parseFrom(s.Stmt.From)
	if err != nil {
		return err
	}

	table, ok := findTableByName(tname)
	if !ok {
		return ErrNotFoundTable
	}
	s.table = table

	err = s.parseCols(s.Stmt.SelectExprs)
	if err != nil {
		return err
	}

	if s.Stmt.Where != nil {
		err = util.PanicHandler(func() {
			s.where = s.parseWhere(s.Stmt.Where)
		})
		if err != nil {
			return err
		}
	}

	if s.Stmt.GroupBy != nil {
		return ErrGroupBy
	}
	if s.Stmt.Having != nil {
		return ErrHaving
	}
	if s.Stmt.OrderBy != nil {
		return ErrOrderBy
	}
	if s.Stmt.Limit != nil {
		err = util.PanicHandler(func() {
			s.offset, s.rows = s.parseLimit(s.Stmt.Limit)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// parseFrom returns a table name.
func (s *SelectParser) parseFrom(froms sqlparser.TableExprs) (string, error) {
	if len(froms) < 1 {
		panic("len(stmt.From) == 0")
	}
	if len(froms) > 1 {
		return "", ErrJoin
	}

	switch from := froms[0].(type) {
	case *sqlparser.AliasedTableExpr:
		if from.As.String() != "" {
			return "", ErrDBQualifier
		}
		switch table := from.Expr.(type) {
		case sqlparser.TableName:
			if table.Qualifier.String() != "" {
				return "", ErrDBQualifier
			}
			return table.Name.String(), nil
		case *sqlparser.Subquery:
			return "", ErrSubquery
		default:
			panic(fmt.Errorf("bug table=%T", table))
		}
	case *sqlparser.ParenTableExpr:
		return "", ErrSubquery
	case *sqlparser.JoinTableExpr:
		return "", ErrJoin
	default:
		panic(fmt.Errorf("bug from=%T", from))
	}
}

// parseSelectCols parses columns and sets the SelectParser.field field.
func (s *SelectParser) parseCols(cols sqlparser.SelectExprs) error {
	var fields []Field
	for i := range cols {
		switch col := cols[i].(type) {
		case *sqlparser.StarExpr:
			if !col.TableName.Qualifier.IsEmpty() {
				return ErrDBQualifier
			}

			useAlias := true
			t := s.table
			tname := s.table.Name
			if !col.TableName.Name.IsEmpty() {
				useAlias = false
				tname = col.TableName.Name.String()
				var ok bool
				t, ok = findTableByName(tname)
				if !ok {
					return fmt.Errorf("not found \"%s\" table", tname)
				}
			}
			// "*" のみが指定されているため、テーブルの全ての列を追加。
			for i := range t.Fields {
				f := Field{
					Table: tname,
					Name:  t.Fields[i],
				}
				if useAlias {
					// テーブル名を削る
					f.AliasName = t.Fields[i]
				}
				fields = append(fields, f)
			}
		case *sqlparser.AliasedExpr:
			if col.As.String() != "" {
				return ErrColumnAlias
			}
			switch col := col.Expr.(type) {
			case *sqlparser.ColName:
				tname := col.Qualifier.Name.String()
				if tname == "" {
					tname = s.table.Name
				}
				f := Field{
					Table: tname,
					Name:  col.Name.String(),
				}
				if col.Qualifier.Name.IsEmpty() {
					f.AliasName = col.Name.String()
				}
				fields = append(fields, f)
			default:
				return ErrColumnList
			}
		}
	}

	// 全てのフィールドが存在するかチェック。
	// 存在しないフィールドを指定したときは、エラーを返す。
	for _, field := range fields {
		var found bool
		if field.Table == s.table.Name {
			found = s.table.HasField(field.Name)
		} else if field.Table == s.table.ImplictJoin {
			t, ok := findTableByName(field.Table)
			if ok {
				found = t.HasField(field.Name)
			}
		}
		if !found {
			return fmt.Errorf("not found \"%s\" column", field.String())
		}
	}
	s.fields = fields
	return nil
}
func (s *SelectParser) parseWhere(where *sqlparser.Where) SqlAny {
	if where.Type != sqlparser.WhereStr {
		panic(fmt.Errorf("bug %#v", where))
	}
	return s.parseWhereExpr(where.Expr)
}

// parseWhereExpr parses expressions and sets comparesion function to SelectParser.where field.
func (s *SelectParser) parseWhereExpr(expr sqlparser.Expr) SqlAny {
	switch expr := expr.(type) {
	case *sqlparser.AndExpr:
		return &AndOp{
			Left:  s.parseWhereExpr(expr.Left),
			Right: s.parseWhereExpr(expr.Right),
		}
	case *sqlparser.OrExpr:
		return &OrOp{
			Left:  s.parseWhereExpr(expr.Left),
			Right: s.parseWhereExpr(expr.Right),
		}
	case *sqlparser.NotExpr:
		return &NotOp{
			Expr: s.parseWhereExpr(expr.Expr),
		}
	case *sqlparser.ParenExpr:
		return s.parseWhereExpr(expr.Expr)
	case *sqlparser.ComparisonExpr:
		//if !supportedCompOps.Contains(expr.Operator) {
		//	panic(fmt.Errorf("unsupported operator: %s", expr.Operator))
		//}
		return &CompOp{
			Operator: expr.Operator,
			Left:     s.parseWhereExpr(expr.Left),
			Right:    s.parseWhereExpr(expr.Right),
		}
	case *sqlparser.RangeCond:
		rangeOp := &RangeOp{
			Left: s.parseWhereExpr(expr.Left),
			From: s.parseWhereExpr(expr.From),
			To:   s.parseWhereExpr(expr.To),
		}
		switch expr.Operator {
		case sqlparser.BetweenStr:
			return rangeOp
		case sqlparser.NotBetweenStr:
			return &NotOp{
				Expr: rangeOp,
			}
		default:
			panic(fmt.Errorf("bug: RangeCond.Operator=%s", expr.Operator))
		}
	case *sqlparser.SQLVal:
		switch expr.Type {
		case sqlparser.StrVal:
			return SqlString(string(expr.Val))
		case sqlparser.IntVal:
			val, err := strconv.ParseInt(string(expr.Val), 10, 64)
			if err != nil {
				panic(err)
			}
			return SqlBigInt(val)
		default:
			// TODO
			panic("todo")
		}
	case *sqlparser.ColName:
		table := s.table.Name
		if expr.Qualifier.Name.String() != "" {
			table = expr.Qualifier.Name.String()
		}
		return &SqlField{
			Field: Field{
				Table: table,
				Name:  expr.Name.String(),
			},
		}
	case *sqlparser.IntervalExpr:
		// TODO
		panic("todo")

	case *sqlparser.FuncExpr:
		if !expr.Qualifier.IsEmpty() {
			panic(ErrFunctionQualifier)
		}
		if expr.Distinct {
			panic(ErrDistinct)
		}

		var found bool
		var sqlfunc SqlFunc
		for i := range funcs {
			if expr.Name.EqualString(funcs[i].Name) {
				found = true
				sqlfunc = funcs[i]
			}
		}
		if !found {
			panic(fmt.Errorf("not found %s function", expr.Name.String()))
		}

		// 関数の引数をパースするは、補完されるテーブル名を SqlFunc で定義されたテーブル名に変更する。
		parser := s
		if sqlfunc.Table != "" {
			table, ok := findTableByName(sqlfunc.Table)
			if !ok {
				panic(fmt.Errorf("not found %s table", sqlfunc.Table))
			}
			parser = &SelectParser{
				table: table,
			}
		}

		var fnargs []SqlAny
		for _, arg := range expr.Exprs {
			fnargs = append(fnargs, parser.parseSelectExpr(arg))
		}
		return sqlfunc.Parse(fnargs...)
	default:
		panic("bug")
	}
	return nil
}
func (s *SelectParser) parseSelectExpr(expr sqlparser.SelectExpr) SqlAny {
	switch expr := expr.(type) {
	case *sqlparser.StarExpr:
		panic("not allowed")
	case *sqlparser.AliasedExpr:
		return s.parseWhereExpr(expr.Expr)
	default:
		panic("bug")
	}
}
func (s *SelectParser) parseLimit(limitObj *sqlparser.Limit) (offset, rows int64) {
	if limitObj == nil {
		return
	}

	var err error
	switch o := limitObj.Offset.(type) {
	case nil:
	case *sqlparser.SQLVal:
		if o.Type != sqlparser.IntVal {
			panic(fmt.Errorf("\"%s\" type is not allowed in offset", string(o.Type)))
		}
		offset, err = strconv.ParseInt(string(o.Val), 10, 64)
		if err != nil {
			panic(errors.Wrap(err, "offset"))
		}
	default:
		panic(fmt.Errorf("bug: limitObj.Offset=%T", o))
	}

	switch r := limitObj.Rowcount.(type) {
	case *sqlparser.SQLVal:
		if r.Type != sqlparser.IntVal {
			panic(fmt.Errorf("\"%s\" type is not allowed in rowcount", string(r.Type)))
		}
		rows, err = strconv.ParseInt(string(r.Val), 10, 64)
		if err != nil {
			panic(errors.Wrap(err, "rowscount"))
		}
	default:
		panic(fmt.Errorf("bug: limitObj.Rowcount=%T", r))
	}
	return
}
func (s *SelectParser) Cols() []Field {
	return s.fields
}
func (s *SelectParser) ColNames() []string {
	names := make([]string, len(s.fields))
	for i := range names {
		names[i] = s.fields[i].String()
	}
	return names
}
func (s *SelectParser) From() string {
	return s.table.Name
}
func (s *SelectParser) Where() SqlAny {
	return s.where
}
func (s *SelectParser) Limit() (offset, rows int64) {
	if s.Stmt.Limit == nil {
		return 0, 0
	}
	return s.offset, s.rows
}

// ParseSelect parses the SELECT statement.
func ParseSelect(sql string) (*SelectParser, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, err
	}

	switch stmt := stmt.(type) {
	case *sqlparser.Select:
		sel := &SelectParser{
			Stmt: stmt,
		}
		err = sel.Parse()
		if err != nil {
			return nil, err
		}
		return sel, nil
	default:
		return nil, ErrUnsupportedStmt
	}
}
