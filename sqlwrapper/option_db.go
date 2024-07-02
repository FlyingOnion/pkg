package sqlwrapper

type optionDB struct {
	ping    bool
	onNull  Strategy
	dialect Dialect
	vc      ValueConverter
}

type OptionDB func(opt *optionDB)

// NoPing 指定时，初始化以后不进行 Ping 操作。
func NoPing() OptionDB {
	return func(opt *optionDB) { opt.ping = false }
}

// WithStrategyOnNull 设置当数据库中的值为 NULL 时对目标变量的行为。
//
//	DoNothing // 保留目标变量的原值，跳过后续的解析，直接开始解析下一列。
//	SetZero   // 将目标变量设为它类型的 0 值，如整型则设为 0，字符串设为 ""，interface{} 设为 nil
//	Continue  // 交给 ValueConverter 处理
func WithStrategyOnNull(s Strategy) OptionDB {
	return func(opt *optionDB) {
		if s == DoNothing || s == SetZero {
			opt.onNull = s
			return
		}
		opt.onNull = Continue
	}
}

func WithDialect(d Dialect) OptionDB {
	return func(opt *optionDB) { opt.dialect = d }
}

func WithValueConverter(converter ValueConverter) OptionDB {
	return func(opt *optionDB) { opt.vc = converter }
}
