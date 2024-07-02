package sqlwrapper

// ValueGroup 可以用于 Where 参数中需要使用 in 的情景。
//
//	db.QueryMultiple(ctx, &list,
//	  Where("age in ?", ValueGroup(20, 30)),
//	)
//
// 多列 where in 语句（请注意不是所有数据库都支持这种语句，目前仅 MySQL 和 Postgresql 测通）。通用方法大多是使用 inner join。
//
//	// Note that not all database supports multiple-column where-in clause
//	db.QueryMultiple(ctx, &list,
//	  Where("(gender, age) in ?",
//	    ValueGroup(
//	      ValueGroup(1, 20),
//	      ValueGroup(0, 30),
//	    ),
//	  ),
//	)
func ValueGroup(vs ...interface{}) valuegroup { return valuegroup(vs) }

type valuegroup []interface{}

func (vg valuegroup) AppendToSqlCtx(ctx *SqlCtx) error {
	if len(vg) == 0 {
		return nil
	}
	ctx.WriteByte('(')
	for i, v := range vg {
		if i > 0 {
			ctx.WriteString(", ")
		}
		if ctxAppender, ok := v.(SqlCtxAppender); ok {
			err := ctxAppender.AppendToSqlCtx(ctx)
			if err != nil {
				return err
			}
			continue
		}
		ctx.NextPlaceholder(v)
	}
	ctx.WriteByte(')')
	return nil
}
