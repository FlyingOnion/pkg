package sqlwrapper

type (
	OptionQuerySingle interface {
		applyToOptionQuerySingle(opt *optQuerySingle)
	}

	OptionQueryMultiple interface {
		applyToOptionQueryMultiple(opt *optQueryMultiple)
	}

	OptionQuery interface {
		OptionQuerySingle
		OptionQueryMultiple
	}

	OptionTable interface {
		applyToOptionTable(opt *optTable)
	}

	OptionJoin interface {
		applyToOptionJoin(opt *optJoin)
	}

	OptionTableAndJoin interface {
		OptionTable
		OptionJoin
	}

	OptionWhere interface {
		OptionQuery
		OptionDelete
	}

	OptionExec interface {
		applyToOptionExec(opt *optExec)
	}

	OptionDelete interface {
		applyToOptionDelete(opt *optDelete)
	}
)

type optQuery struct {
	selectColumns  []string
	table          optTable
	whereClause    string
	whereArgs      []interface{}
	groupByColumns []string
	havingClause   string
	havingArgs     []interface{}
	orderByColumns []string
	limit          uint64
	offset         uint64
	isSubQuery     bool
}

func (o optQuery) applyToOptionTable(t *optTable) { t.table = o }
func (o optQuery) applyToOptionJoin(j *optJoin)   { j.table = o }
func (o optQuery) AppendToSqlCtx(ctx *SqlCtx) (err error) {
	if o.isSubQuery {
		ctx.WriteByte('(')
	}
	err = ctx.selectFromTable(o.selectColumns, o.table)
	if err != nil {
		return
	}
	err = ctx.where(o.whereClause, o.whereArgs...)
	if err != nil {
		return
	}
	ctx.groupBy(o.groupByColumns...)
	err = ctx.having(o.havingClause, o.havingArgs...)
	if err != nil {
		return
	}
	switch ctx.driver {
	case "oci8", "oracle":
		ctx.orderBy(o.orderByColumns...)
		ctx.offsetFetchNextRows(o.offset, o.limit)
	case "mssql", "sqlserver":
		if len(o.orderByColumns) == 0 {
			ctx.WriteString(" order by 1")
		}
		ctx.offsetFetchNextRows(o.offset, o.limit)
	case "mysql":
		if o.limit == 0 && o.offset > 0 {
			o.limit = MySQLUnlimit
		}
		fallthrough
	default:
		ctx.orderBy(o.orderByColumns...)
		ctx.limitOffset(o.limit, o.offset)
	}
	if o.isSubQuery {
		ctx.WriteByte(')')
	}
	return
}

type optQuerySingle struct {
	optQuery
	unused map[string]interface{}
}

type optQueryMultiple struct {
	optQuery
}

type optExec struct {
	columns        []string
	includingZeros bool
}

type optDelete struct {
	whereClause string
	whereArgs   []interface{}
}

type (
	optSelect struct{ columns []string }
	optTable  struct {
		table SqlCtxAppender
		alias string
		joins []optJoin
	}
	optSingleTable struct {
		table string
	}
	optJoin struct {
		table SqlCtxAppender
		alias string
		// joinType is one of "inner join", "left join", "right join" and "outer join"
		joinType string
		// condType is one of "on" and "using"
		condType string
		// conds:
		//  ["foo.id = bar.foo_id", "..."] // when condType is "on"
		//  ["foo_id", "foo_name", "..."]  // when condType is "using"
		conds []string
	}
	optJoinCondition struct {
		condType string
		conds    []string
	}
	optAlias string
	optWhere struct {
		clause string
		args   []interface{}
	}

	optGroupBy struct {
		columns []string
	}

	optHaving struct {
		clause string
		args   []interface{}
	}
	optOrderBy struct {
		columns []string
	}
	optLimit  uint64
	optOffset uint64
	optUnused map[string]interface{}

	optIncludingZeros struct{}
	optColumns        struct {
		columns []string
	}
)

func (o optSelect) applyToOptionQuerySingle(q *optQuerySingle)     { q.selectColumns = o.columns }
func (o optSelect) applyToOptionQueryMultiple(q *optQueryMultiple) { q.selectColumns = o.columns }

func (o optTable) applyToOptionQuerySingle(q *optQuerySingle)     { q.table = o }
func (o optTable) applyToOptionQueryMultiple(q *optQueryMultiple) { q.table = o }

func (o optSingleTable) applyToOptionTable(f *optTable) { f.table = o }
func (o optSingleTable) applyToOptionJoin(j *optJoin)   { j.table = o }
func (o optSingleTable) AppendToSqlCtx(ctx *SqlCtx) error {
	ctx.WriteQuotedString(o.table)
	return nil
}

func (o optAlias) applyToOptionTable(t *optTable) { t.alias = string(o) }
func (o optAlias) applyToOptionJoin(j *optJoin)   { j.alias = string(o) }

func (o optJoinCondition) applyToOptionJoin(j *optJoin) { j.condType, j.conds = o.condType, o.conds }

func (o optWhere) applyToOptionQuerySingle(q *optQuerySingle) {
	q.whereClause, q.whereArgs = o.clause, o.args
}
func (o optWhere) applyToOptionQueryMultiple(q *optQueryMultiple) {
	q.whereClause, q.whereArgs = o.clause, o.args
}
func (o optWhere) applyToOptionDelete(d *optDelete) { d.whereClause, d.whereArgs = o.clause, o.args }

func (o optGroupBy) applyToOptionQuerySingle(q *optQuerySingle)     { q.groupByColumns = o.columns }
func (o optGroupBy) applyToOptionQueryMultiple(q *optQueryMultiple) { q.groupByColumns = o.columns }

func (o optHaving) applyToOptionQuerySingle(q *optQuerySingle) {
	q.havingClause, q.havingArgs = o.clause, o.args
}
func (o optHaving) applyToOptionQueryMultiple(q *optQueryMultiple) {
	q.havingClause, q.havingArgs = o.clause, o.args
}

func (o optOrderBy) applyToOptionQuerySingle(q *optQuerySingle)     { q.orderByColumns = o.columns }
func (o optOrderBy) applyToOptionQueryMultiple(q *optQueryMultiple) { q.orderByColumns = o.columns }

func (o optOffset) applyToOptionQuerySingle(q *optQuerySingle)     { q.offset = uint64(o) }
func (o optOffset) applyToOptionQueryMultiple(q *optQueryMultiple) { q.offset = uint64(o) }

func (o optUnused) applyToOptionQuerySingle(q *optQuerySingle) { q.unused = o }

func (o optLimit) applyToOptionQueryMultiple(q *optQueryMultiple) { q.limit = uint64(o) }

func (o optJoin) applyToOptionTable(t *optTable) { t.joins = append(t.joins, o) }

func (o optColumns) applyToOptionExec(e *optExec) { e.columns = o.columns }

func (o optIncludingZeros) applyToOptionExec(e *optExec) { e.includingZeros = true }

// Select 可以查询指定的列。
//
//	Select("id", "name")    // select id, name
//	Select("distinct name") // select distinct name
//	Select("count(*)")      // select count(*)
//	Select("1+2")           // select 1+2
//
//	quote := GetDialect(db.Driver())
//	Select(quote("id"), quote("name")) // select `id`, `name`
func Select(columns ...string) OptionQuery {
	return optSelect{columns: columns}
}

// From 可以指定表名。
//
//	From(Table("user"))
func From(options ...OptionTable) OptionQuery {
	o := &optTable{}
	for _, opt := range options {
		opt.applyToOptionTable(o)
	}
	return o
}

func Table(table string) OptionTableAndJoin {
	return optSingleTable{table}
}

func SubQuery(options ...OptionQueryMultiple) OptionTableAndJoin {
	o := optQueryMultiple{}
	for _, opt := range options {
		opt.applyToOptionQueryMultiple(&o)
	}
	o.isSubQuery = true
	return o.optQuery
}

// InnerJoin provides an "inner join" table join option.
func InnerJoin(options ...OptionJoin) OptionTable {
	o := &optJoin{joinType: "inner join"}
	for _, opt := range options {
		opt.applyToOptionJoin(o)
	}
	return o
}

// LeftJoin provides an "left join" table join option.
func LeftJoin(options ...OptionJoin) OptionTable {
	o := &optJoin{joinType: "left join"}
	for _, opt := range options {
		opt.applyToOptionJoin(o)
	}
	return o
}

// RightJoin provides an "right join" table join option.
func RightJoin(options ...OptionJoin) OptionTable {
	o := &optJoin{joinType: "right join"}
	for _, opt := range options {
		opt.applyToOptionJoin(o)
	}
	return o
}

// FullJoin provides an "full join" table join option.
func FullJoin(options ...OptionJoin) OptionTable {
	o := &optJoin{joinType: "full join"}
	for _, opt := range options {
		opt.applyToOptionJoin(o)
	}
	return o
}

func As(alias string) OptionTableAndJoin {
	return optAlias(alias)
}

func On(conds ...string) OptionJoin {
	return optJoinCondition{"on", conds}
}

func Using(conds ...string) OptionJoin {
	return optJoinCondition{"using", conds}
}

func Where(clause string, args ...interface{}) OptionWhere {
	return optWhere{clause: clause, args: args}
}

func GroupBy(columns ...string) OptionQuery {
	return optGroupBy{columns: columns}
}

func Having(clause string, args ...interface{}) OptionQuery {
	return optHaving{clause: clause, args: args}
}

// OrderBy 排序指定的列。
//
//	OrderBy("id")
//	OrderBy("id asc", "name desc")
func OrderBy(columns ...string) OptionQuery {
	return optOrderBy{columns: columns}
}

func Offset(offset uint64) OptionQuery {
	return optOffset(offset)
}

func Limit(limit uint64) OptionQueryMultiple {
	return optLimit(limit)
}

// 当 Query 的数据列数大于目标结构体有效的字段数时，可以使用该 Option 记录结构体字段以外的列。
//
//	type User struct {
//	    ID int
//	    Name string
//	}
//	var m map[string]interface{}
//	db.Query(ctx, &User{},
//	  Select("id", "name", "age"),
//	  RetrieveUnusedValuesTo(m),
//	)
//	fmt.Println(m["age"])
func RetrieveUnusedValuesTo(m map[string]interface{}) OptionQuerySingle {
	return optUnused(m)
}

// WithColumns 可以自定义插入、更新哪些列，columns 为数据库列名。
// 指定该 Option 后依然会检查字段是否为〇值。
func WithColumns(columns ...string) OptionExec {
	return optColumns{columns}
}

// IncludingZeros 设置时，entity 中的〇值字段不会被忽略，将以其类型的〇值传入 db.ExecContext 的参数列表。
func IncludingZeros() OptionExec {
	return optIncludingZeros{}
}
