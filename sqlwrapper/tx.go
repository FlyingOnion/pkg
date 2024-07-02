package sqlwrapper

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

type Tx struct {
	origin *sql.Tx
	db     *Database // still need db fields
}

// RawQuery 封装了 (*sql.Tx).Query 方法。使用方法与 db.RawQuery 基本一致。
//
// 对 sql.Rows 的操作封装到 RowsScanner。若结果只有一行，可以使用 SingleRowScanner。
//
// 结果有多行时建议使用 ScanFn，详见 RowsScanner 注释。
func (tx *Tx) RawQuery(query string, s RowsScanner, args ...interface{}) (err error) {
	// There is already a ctx in origin. No need to create new ctx here.
	rows, err := tx.origin.Query(query, args...)
	if err != nil {
		return err
	}
	err = s.ScanFrom(rows)
	rows.Close()
	return
}

// RawExec 封装了 (*sql.Tx).Exec 方法，直接返回了 sql.Result 和 error。
func (tx *Tx) RawExec(query string, args ...interface{}) (sql.Result, error) {
	// There is already a ctx in origin. No need to create new ctx here.
	return tx.origin.Exec(query, args...)
}

func (tx *Tx) Query(entity interface{}, options ...OptionQuerySingle) (found bool, err error) {
	if reflect.TypeOf(entity).Kind() != reflect.Ptr {
		err = ErrNotPointer
		return
	}
	db := tx.db
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
	err = tx.RawQuery(ctx.QueryString(),
		StructScanner(entity, sm, db.vc, db.onNull, q.unused, &found),
		ctx.args...)
	return
}

func (tx *Tx) QueryMultiple(es interface{}, options ...OptionQueryMultiple) (err error) {
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
	db := tx.db
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
	err = tx.RawQuery(ctx.QueryString(),
		SliceScanner(es, sm, db.vc, db.onNull, t, isPointer),
		ctx.args...)
	return
}

func (tx *Tx) Insert(e IEntity, options ...OptionExec) (err error) {
	db := tx.db
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
		_, err = tx.RawExec(ctx.QueryString(), ctx.args...)
		return
	}

	pkIndex := sm.pkStructIndex
	if pkIndex == -1 || !v.Field(pkIndex).IsZero() {
		// 没有主键字段，或者有主键字段但不为〇值，则不进行 ID 赋值，执行后直接返回。
		_, err = tx.RawExec(ctx.QueryString(), ctx.args...)
		return
	}

	var newId interface{}
	// 使用 interface{} 类型的原因有二。
	// 1. 使用 query returning 方式返回时 Scan 赋值不会报错；
	// 2. 需要考虑主键是字符串的情况。
	switch db.driver {
	case "pgx", "postgres":
		ctx.WriteString(" returning ").WriteQuotedString(e.PkColumn())
		err = tx.RawQuery(ctx.QueryString(), SingleRowScanner(&newId), ctx.args...)
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
			_, err = tx.RawExec(ctx.QueryString(), ctx.args...)
			return
		}
		ctx.WriteString("; select last_id = convert(bigint, SCOPE_IDENTITY())")
		err = tx.RawQuery(ctx.QueryString(), SingleRowScanner(&newId), ctx.args...)
		if err != nil {
			return
		}
	default:
		var result sql.Result
		result, err = tx.RawExec(ctx.QueryString(), ctx.args...)
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

// Update 更新 e 对应的数据库中的记录。e 的主键字段必须非空。
//
// 只想保存，不想管是插入还是更新的话，可以使用通用方法 Save。
func (tx *Tx) Update(e IEntity, options ...OptionExec) (err error) {
	db := tx.db
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

	_, err = tx.RawExec(ctx.QueryString(), ctx.args...)
	return
}

// Save 将 e 保存到数据库中。当 e 主键字段为空时插入，非空时更新。
func (tx *Tx) Save(e IEntity, options ...OptionExec) error {
	sm, err := tx.db.RegisterType(e)
	if err != nil {
		return err
	}
	v := reflect.ValueOf(e)
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	pkField := v.Field(sm.pkStructIndex)
	if pkField.IsZero() {
		return tx.Insert(e, options...)
	}
	return tx.Update(e, options...)
}

type TransactionStep int8

const (
	StepEnd    TransactionStep = -iota // Transaction is successfully commited
	StepBegin                          // Transaction failed at BeginTx step
	StepRun                            // Transaction failed because of error returned from run
	StepCommit                         // Transaction failed to commit to db
)

// RunTx can run a transaction. To pass *sql.TxOptions as a parameter, use RunTxWithOptions.
//
// TransactionStep represents the last step of transaction before exit. The corresponding error is also returned. Transaction is successfully commited if and only if TransactionStep is StepEnd.
//
//	db.RunTx(func(tx *Tx) (commit bool, err error) {
//	    err := tx.Insert(&foo)
//	    if err != nil {
//	        return false, err
//	    }
//	    err = tx.Insert(&bar)
//	    if err != nil {
//	        return false, err
//	    }
//	    return true, nil
//	})
func (db *Database) RunTx(
	run func(tx *Tx) (commit bool, err error),
) (TransactionStep, error) {
	return db.RunTxWithOptions(run, nil)
}

func (db *Database) RunTxWithOptions(
	run func(tx *Tx) (commit bool, err error),
	txOptions *sql.TxOptions,
) (TransactionStep, error) {
	ctx, cancel := context.WithCancel(db.ctx)
	defer cancel()
	tx, err := db.origin.BeginTx(ctx, txOptions)
	if err != nil {
		return StepBegin, fmt.Errorf(fx1, err)
	}
	defer tx.Rollback()
	commit, err := run(&Tx{tx, db})
	if err != nil {
		return StepRun, err
	}
	if commit {
		err = tx.Commit()
		if err != nil {
			return StepCommit, err
		}
	}
	return StepEnd, nil
}
