package sqlwrapper

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sync"
)

type Database struct {
	driver  string
	dialect Dialect
	origin  *sql.DB

	onNull Strategy
	vc     ValueConverter

	ctxpool sync.Pool

	// ctx is the base of all operations
	ctx    context.Context
	cancel context.CancelFunc
}

func NewDatabase(driver, dsn string, options ...OptionDB) (*Database, error) {
	o := &optionDB{
		ping:   true,
		onNull: DoNothing,
		vc:     Vcie,
	}
	for _, opt := range options {
		opt(o)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	if o.ping {
		if err = db.Ping(); err != nil {
			db.Close()
			return nil, err
		}
	}
	dialect := GetDialect(driver)
	ctx, cancel := context.WithCancel(context.Background())
	d := &Database{
		driver:  driver,
		dialect: dialect,
		origin:  db,
		onNull:  o.onNull,
		vc:      o.vc,
		ctxpool: sync.Pool{
			New: func() interface{} {
				return NewContext(driver, dialect)
			},
		},
		ctx:    ctx,
		cancel: cancel,
	}
	return d, nil
}

func (db *Database) Close() (err error) {
	db.cancel()
	err = db.origin.Close()
	return
}

func (db *Database) Origin() *sql.DB { return db.origin }

func (db *Database) Driver() string { return db.driver }

func (db *Database) newContext() *SqlCtx {
	if ctx, ok := db.ctxpool.Get().(*SqlCtx); ok {
		ctx.Reset()
		return ctx
	}
	return NewContext(db.driver, db.dialect)
}

func (db *Database) recycleContext(ctx *SqlCtx) {
	ctx.Reset()
	db.ctxpool.Put(ctx)
}

func (db *Database) Insert(e IEntity, options ...OptionExec) (err error) {
	sm, err := db.RegisterType(e)
	if err != nil {
		return
	}

	o := &optExec{
		columns: sm.columns,
	}
	for _, opt := range options {
		opt.applyToOptionExec(o)
	}

	ctx := db.newContext()
	defer db.recycleContext(ctx)
	nColumns := len(o.columns)
	args := make([]interface{}, 0, nColumns)
	v := reflect.ValueOf(e)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 构建 sql 语句
	ctx.WriteString("insert into ").
		WriteQuotedString(e.TableName()).
		WriteQuotedString(" (")
	for _, column := range o.columns {
		fm, ok := sm.columnFieldMap[column]
		if !ok {
			// 没找到该列，说明调用时传入的 WithColumns 中列名可能写错了。
			return fmt.Errorf(f5, column)
		}
		field := v.Field(fm.index)
		if field.IsZero() && !o.includingZeros {
			// 没有指定 includingZeros 时跳过默认〇值的字段
			continue
		}
		if len(args) > 0 {
			ctx.WriteString(", ")
		}
		ctx.WriteQuotedString(column)
		args = append(args, field.Interface())
	}

	ctx.WriteString(") values (")
	for i, a := range args {
		if i > 0 {
			ctx.WriteString(", ")
		}
		ctx.NextPlaceholder(a)
	}
	ctx.WriteByte(')')

	// 传入的是个结构体而非指针，返回了 id 也无法赋值。
	// 直接执行后结束。
	if reflect.TypeOf(e).Kind() != reflect.Ptr {
		_, err = db.RawExec(ctx.QueryString(), ctx.args...)
		return
	}

	pkIndex := sm.pkStructIndex
	if pkIndex == -1 || !v.Field(pkIndex).IsZero() {
		// 没有主键字段，或者有主键字段但不为〇值，则不进行 ID 赋值，执行后直接返回。
		_, err = db.RawExec(ctx.QueryString(), ctx.args...)
		return
	}

	var newId interface{}
	// 使用 interface{} 类型的原因有二。
	// 1. 使用 query returning 方式返回时 Scan 赋值不会报错；
	// 2. 需要考虑主键是字符串的情况。
	switch db.driver {
	case "pgx", "postgres":
		ctx.WriteString(" returning ").WriteQuotedString(e.PkColumn())
		err = db.RawQuery(ctx.QueryString(), SingleRowScanner(&newId), ctx.args...)
		if err != nil {
			return
		}
		// t := reflect.TypeOf(newId)
		// log.LogInfo(t.Kind(), t.String())

		// 不能确定 returning 的 newId 一定是 int 类型（有可能是 string），需要判断。
		// 除了 string/uuid -> int 不行以外其他都可以转。
		if !reflect.TypeOf(newId).ConvertibleTo(sm.pkType) {
			return
		}
	case "mssql", "sqlserver":
		if sm.pkType.Kind() == reflect.String {
			// sqlserver 没有办法返回字符串类型主键，直接执行后返回。
			_, err = db.RawExec(ctx.QueryString(), ctx.args...)
			return
		}
		ctx.WriteString("; select last_id = convert(bigint, SCOPE_IDENTITY())")
		err = db.RawQuery(ctx.QueryString(), SingleRowScanner(&newId), ctx.args...)
		if err != nil {
			return
		}
	default:
		var result sql.Result
		result, err = db.RawExec(ctx.QueryString(), ctx.args...)
		if err != nil {
			return
		}
		var err1 error
		// 获取新记录的 ID。
		newId, err1 = result.LastInsertId()
		if err1 != nil {
			// 不支持 LastInsertId 方法，直接返回。
			// TODO: write a warning log
			return
		}
	}

	pkField := v.Field(pkIndex)
	pkField.Set(reflect.ValueOf(newId).Convert(sm.pkType))
	return
}

func (db *Database) Query(entity interface{}, options ...OptionQuerySingle) (found bool, err error) {
	if reflect.TypeOf(entity).Kind() != reflect.Ptr {
		err = ErrNotPointer
		return
	}
	sm, err := db.RegisterType(entity)
	if err != nil {
		return
	}

	table := ""
	if e, ok := entity.(IEntity); ok {
		table = e.TableName()
	}

	q := &optQuerySingle{
		optQuery: optQuery{
			selectColumns: sm.columns,
			table: optTable{
				table: optSingleTable{table},
			},
			limit: 1,
		},
	}
	for _, opt := range options {
		opt.applyToOptionQuerySingle(q)
	}

	ctx := db.newContext()
	defer db.recycleContext(ctx)
	err = q.optQuery.AppendToSqlCtx(ctx)
	if err != nil {
		return
	}
	// fmt.Println(ctx.QueryString(), ctx.args)
	err = db.RawQuery(ctx.QueryString(),
		StructScanner(entity, sm, db.vc, db.onNull, q.unused, &found),
		ctx.args...)
	return
}

func (db *Database) QueryMultiple(es interface{}, options ...OptionQueryMultiple) (err error) {
	t := reflect.TypeOf(es)
	if t.Kind() != reflect.Ptr {
		return ErrNotPointer
	}
	t = t.Elem()
	if t.Kind() != reflect.Slice {
		return ErrElemNotSlice
	}
	t = t.Elem()

	isPointer := t.Kind() == reflect.Ptr
	if isPointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return ErrElemNotStruct
	}

	entity := reflect.New(t).Interface()
	sm, err := db.RegisterType(entity)
	if err != nil {
		return
	}

	table := ""
	if e, ok := entity.(IEntity); ok {
		table = e.TableName()
	}

	q := &optQueryMultiple{
		optQuery: optQuery{
			selectColumns: sm.columns,
			table: optTable{
				table: optSingleTable{table},
			},
		},
	}
	for _, opt := range options {
		opt.applyToOptionQueryMultiple(q)
	}

	ctx := db.newContext()
	defer db.recycleContext(ctx)
	err = q.optQuery.AppendToSqlCtx(ctx)
	if err != nil {
		return
	}
	// fmt.Println(ctx.QueryString(), ctx.args)
	err = db.RawQuery(ctx.QueryString(),
		SliceScanner(es, sm, db.vc, db.onNull, t, isPointer),
		ctx.args...)
	return
}

func (db *Database) UpdateMap(ctx context.Context, updates map[string]interface{}) error {
	// TODO
	return nil
}

func (db *Database) BatchUpdate(ctx context.Context, es []interface{}) error {
	// TODO
	return nil
}

// Update 更新 e 对应的数据库中的记录。e 的主键字段必须非空。
//
// 只想保存，不想管是插入还是更新的话，可以使用通用方法 Save。
func (db *Database) Update(e IEntity, options ...OptionExec) (err error) {
	sm, err := db.RegisterType(e)
	if err != nil {
		return
	}

	v := reflect.ValueOf(e)
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	pkField := v.Field(sm.pkStructIndex)
	if pkField.IsZero() {
		err = fmt.Errorf(f4, e.PkColumn())
		return
	}

	o := &optExec{columns: sm.columns}
	for _, opt := range options {
		opt.applyToOptionExec(o)
	}

	nColumns := len(o.columns)
	if nColumns == 0 {
		// do nothing and return
		return
	}
	ctx := db.newContext()
	defer db.recycleContext(ctx)

	ctx.WriteString("update ").
		WriteQuotedString(e.TableName()).
		WriteString(" set ")

	for _, column := range o.columns {
		if column == e.PkColumn() {
			// 主键字段放 where 子句里面，set 里面不用填
			continue
		}
		fm, ok := sm.columnFieldMap[column]
		if !ok {
			// 没找到该列，说明调用时传入的 WithColumns 中列名可能写错了。
			return fmt.Errorf(f5, column)
		}
		field := v.Field(fm.index)
		if field.IsZero() && !o.includingZeros {
			// 没有指定 includingZeros 时跳过默认〇值的字段
			continue
		}
		if len(ctx.args) > 0 {
			ctx.WriteString(", ")
		}
		ctx.WriteQuotedString(column).
			WriteString(" = ").
			NextPlaceholder(field.Interface())
	}

	ctx.WriteString(" where ").
		WriteQuotedString(e.PkColumn()).
		WriteString(" = ").
		NextPlaceholder(pkField.Interface())

	_, err = db.RawExec(ctx.QueryString(), ctx.args...)
	return
}

// Save 将 e 保存到数据库中。当 e 主键字段为空时插入，非空时更新。
func (db *Database) Save(e IEntity, options ...OptionExec) error {
	sm, err := db.RegisterType(e)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(e)
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	pkField := v.Field(sm.pkStructIndex)
	if pkField.IsZero() {
		return db.Insert(e, options...)
	}
	return db.Update(e, options...)
}

func (db *Database) Delete(table string, options ...OptionDelete) (err error) {
	d := &optDelete{}
	for _, opt := range options {
		opt.applyToOptionDelete(d)
	}
	ctx := db.newContext()
	defer db.recycleContext(ctx)

	ctx.WriteString("delete from ").
		WriteQuotedString(table)
	err = ctx.where(d.whereClause, d.whereArgs...)
	if err != nil {
		return
	}

	_, err = db.RawExec(ctx.QueryString(), ctx.args...)
	return
}

// RawExec 封装了 (*sql.DB).ExecContext 方法，直接返回了 sql.Result 和 error。
func (db *Database) RawExec(query string, args ...interface{}) (sql.Result, error) {
	return db.origin.ExecContext(db.ctx, query, args...)
}

// RawQuery 封装了 (*sql.DB).QueryContext 方法。
//
// 对 sql.Rows 的操作封装到 RowsScanner。若结果只有一行，可以使用 SingleRowScanner。
//
// 结果有多行时建议使用 ScanFn，详见 RowsScanner 注释。
func (db *Database) RawQuery(query string, s RowsScanner, args ...interface{}) (err error) {
	rows, err := db.origin.QueryContext(db.ctx, query, args...)
	if err != nil {
		return err
	}
	err = s.ScanFrom(rows)
	rows.Close()
	return
}

func (db *Database) Quote(s string) string { return db.dialect.Quote(s) }
