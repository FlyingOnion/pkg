package sqlwrapper

import (
	"strings"

	"sync/atomic"
)

const (
	bsz = 64
	asz = 16
)

type SqlCtx struct {
	phid    uint64
	buf     []byte
	args    []interface{}
	driver  string
	dialect Dialect
}

func NewContext(driver string, dialect Dialect) *SqlCtx {
	return &SqlCtx{
		buf:     make([]byte, 0, bsz),
		args:    make([]interface{}, 0, asz),
		driver:  driver,
		dialect: dialect,
	}
}

type SqlCtxAppender interface {
	AppendToSqlCtx(ctx *SqlCtx) error
}

func (ctx *SqlCtx) WriteString(str string) *SqlCtx {
	ctx.buf = append(ctx.buf, str...)
	return ctx
}

func (ctx *SqlCtx) WriteByte(b byte) *SqlCtx {
	ctx.buf = append(ctx.buf, b)
	return ctx
}

func (ctx *SqlCtx) WriteBytes(bs []byte) *SqlCtx {
	ctx.buf = append(ctx.buf, bs...)
	return ctx
}

func (ctx *SqlCtx) WriteQuotedString(str string) *SqlCtx {
	ctx.buf = append(ctx.buf, ctx.dialect.Quote(str)...)
	return ctx
}

func (ctx *SqlCtx) NextPlaceholder(arg interface{}) *SqlCtx {
	ctx.WriteString(ctx.dialect.HoldPlace(atomic.AddUint64(&ctx.phid, 1)))
	ctx.args = append(ctx.args, arg)
	return ctx
}

func (ctx *SqlCtx) QueryString() string { return string(ctx.buf) }

func (ctx *SqlCtx) Arguments() []interface{} { return ctx.args }

func (ctx *SqlCtx) Reset() {
	ctx.buf = ctx.buf[:0]
	ctx.args = ctx.args[:0]
	ctx.phid = 0
}

func (ctx *SqlCtx) selectFromTable(columns []string, table optTable) (err error) {
	ctx.WriteString("select ")
	if len(columns) == 0 {
		ctx.WriteByte('*')
	} else {
		ctx.WriteString(strings.Join(columns, ", "))
	}
	ctx.WriteString(" from ")
	err = table.table.AppendToSqlCtx(ctx)
	if err != nil {
		return
	}
	if len(table.alias) > 0 {
		ctx.WriteString(" as ").WriteQuotedString(table.alias)
	}
	for _, join := range table.joins {
		ctx.WriteByte(' ').
			WriteString(join.joinType).
			WriteByte(' ')
		err = join.table.AppendToSqlCtx(ctx)
		if err != nil {
			return
		}
		if len(join.alias) > 0 {
			ctx.WriteString(" as ").WriteQuotedString(join.alias)
		}
		if len(join.condType) > 0 && len(join.conds) > 0 {
			switch join.condType {
			case "on":
				ctx.WriteString(" on ").WriteString(join.conds[0])
				for _, cond := range join.conds[1:] {
					ctx.WriteString(" and ").WriteString(cond)
				}
			case "using":
				ctx.WriteString(" using (").WriteString(join.conds[0])
				for _, cond := range join.conds[1:] {
					ctx.WriteString(", ").WriteString(cond)
				}
				ctx.WriteByte(')')
			default:
				err = ErrInvalidJoinCondType
				return
			}
		}
	}
	return
}

func (ctx *SqlCtx) clauseWithArgs(clause string, args ...interface{}) error {
	nph, nWhereArgs := 0, len(args)
	i := 0
	for j := 0; j < len(clause); {
		if clause[j] != '?' {
			j++
			continue
		}
		ctx.WriteString(clause[i:j])
		if nph >= nWhereArgs {
			// 占位符数量比参数数量多
			return ErrNotEnoughArgs
		}

		arg := args[nph]
		// 写入占位符，将参数添加到列表
		if ctxAppender, ok := arg.(SqlCtxAppender); ok {
			// 参数组 valuegroup 和子查询需要实现该接口
			err := ctxAppender.AppendToSqlCtx(ctx)
			if err != nil {
				return err
			}
		} else {
			ctx.NextPlaceholder(arg)
		}
		j++
		i = j
		nph++
	}

	if nph < nWhereArgs {
		// 占位符数量比参数数量少
		return ErrTooManyArgs
	}
	ctx.WriteString(clause[i:])
	return nil
}

func (ctx *SqlCtx) where(clause string, args ...interface{}) error {
	if len(clause) == 0 {
		return nil
	}
	return ctx.WriteString(" where ").clauseWithArgs(clause, args...)
}

func (ctx *SqlCtx) groupBy(columns ...string) {
	if len(columns) > 0 {
		ctx.WriteString(" group by ")
		for i, col := range columns {
			if i > 0 {
				ctx.WriteString(", ")
			}
			ctx.WriteQuotedString(col)
		}
	}
}

func (ctx *SqlCtx) having(clause string, args ...interface{}) error {
	if len(clause) == 0 {
		return nil
	}
	return ctx.WriteString(" having ").clauseWithArgs(clause, args...)
}

func (ctx *SqlCtx) orderBy(columns ...string) {
	if len(columns) > 0 {
		ctx.WriteString(" order by ")
		for i, col := range columns {
			if i > 0 {
				ctx.WriteString(", ")
			}
			if space := strings.IndexByte(col, ' '); space == -1 {
				ctx.WriteQuotedString(col)
			} else {
				ctx.WriteQuotedString(col[:space]).WriteString(col[space:])
			}
		}
	}
}

// Yeah. It's a "feature" of MySQL. See https://dev.mysql.com/doc/refman/8.0/en/select.html
const MySQLUnlimit uint64 = 18446744073709551615

func (ctx *SqlCtx) limitOffset(limit, offset uint64) {
	if limit > 0 {
		ctx.WriteString(" limit ").NextPlaceholder(limit)
	}
	if offset > 0 {
		ctx.WriteString(" offset ").NextPlaceholder(offset)
	}
}

func (ctx *SqlCtx) offsetFetchNextRows(offset, limit uint64) {
	if offset > 0 {
		ctx.WriteString(" offset ").NextPlaceholder(offset).WriteString(" rows")
	}
	if limit > 0 {
		ctx.WriteString(" fetch next ").NextPlaceholder(limit).WriteString(" rows only")
	}
}
