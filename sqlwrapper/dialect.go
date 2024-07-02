package sqlwrapper

type Dialect interface {
	ColumnNameConverter
	Placeholder
	Quoter
}

type CustomDialect struct {
	ColumnNameConverter
	Placeholder
	Quoter
}

var (
	defaultDialect = CustomDialect{Snake, QuestionMark, NoQuotes}
	mysql          = CustomDialect{Snake, QuestionMark, Backticks}
	sqlite         = CustomDialect{Snake, QuestionMark, Backticks}
	postgresql     = CustomDialect{Snake, Dollar_I, DoubleQuotes}
	sqlserver      = CustomDialect{Snake, At_P_I, Brackets}
	oracle         = CustomDialect{UpperSnake, Colon_I, DoubleQuotes}
)

var (
	dialectMap = map[string]Dialect{
		"sqlite":    sqlite,
		"sqlite3":   sqlite,
		"mysql":     mysql,
		"oracle":    oracle,
		"oci8":      oracle,
		"pgx":       postgresql,
		"postgres":  postgresql,
		"mssql":     sqlserver,
		"sqlserver": sqlserver,
	}
)

func GetDialect(driver string) Dialect {
	if d, ok := dialectMap[driver]; ok {
		return d
	}
	return defaultDialect
}

// RegisterDialect helps user register their own dialects.
//
// Builtin dialects should not be registered again.
//
//	// builtin dialects:
//	// sqlite, sqlite3, mysql, oracle, oci8, pgx, postgres, mssql, sqlserver
func RegisterDialect(driver string, dialect Dialect) error {
	if _, ok := dialectMap[driver]; ok {
		return ErrDialectAlreadyRegistered
	}
	dialectMap[driver] = dialect
	return nil
}
