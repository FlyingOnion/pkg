package sqlwrapper

import "strings"

// ColumnNameConverter 指定字段名到列名转换方法，当 entity 字段没有指定 tag 时将用该方法转换。
//
// ColumnNameConverter specifies how field name is converted to column name.
//
//	// available converters:
//	Snake      // sqlite, mysql and postgresql default
//	Camel      // mssql default
//	UpperSnake // oracle default
//	AsIs       // use field name directly
//	ToUpper
//	ToLower
type ColumnNameConverter interface {
	Convert(colName string) string
}

type ColumnNameConvertFunc func(s string) string

func (fn ColumnNameConvertFunc) Convert(colName string) string { return fn(colName) }

var (
	// 保持原样
	AsIs ColumnNameConverter = ColumnNameConvertFunc(func(s string) string { return s })
	// 转化为全小写形式
	ToLower ColumnNameConverter = ColumnNameConvertFunc(func(s string) string { return strings.ToLower(s) })
	// 转化为全大写形式
	ToUpper ColumnNameConverter = ColumnNameConvertFunc(func(s string) string { return strings.ToUpper(s) })

	// UpperSnake 转换为全大写下划线形式
	UpperSnake ColumnNameConverter = ColumnNameConvertFunc(func(s string) string {
		n := len(s)
		if n == 0 {
			return ""
		}
		out := make([]byte, 0, n+n/2)

		for i := 0; i < n-2; i++ {
			out = append(out, toUpper(s[i]))
			if isUpper(s[i+1]) && (isLower(s[i]) || isUpper(s[i]) && isLower(s[i+2])) {
				out = append(out, '_')
			}
		}
		if n >= 2 {
			out = append(out, toUpper(s[n-2]))
			if isLower(s[n-2]) && isUpper(s[n-1]) {
				out = append(out, '_')
			}
		}
		out = append(out, toUpper(s[n-1]))
		return string(out)
	})

	// Snake 将字符串转换为下划线形式。该方法对 github.com/azer/snakecase 的实现进行了修改。
	Snake ColumnNameConverter = ColumnNameConvertFunc(func(s string) string {
		n := len(s)
		if n == 0 {
			return ""
		}
		out := make([]byte, 0, n+n/2)

		for i := 0; i < n-2; i++ {
			out = append(out, toLower(s[i]))
			if isUpper(s[i+1]) && (isLower(s[i]) || isUpper(s[i]) && isLower(s[i+2])) {
				out = append(out, '_')
			}
		}
		if n >= 2 {
			out = append(out, toLower(s[n-2]))
			if isLower(s[n-2]) && isUpper(s[n-1]) {
				out = append(out, '_')
			}
		}
		out = append(out, toLower(s[n-1]))
		return string(out)
	})

	// Camel 将字符串转换为驼峰形式。
	// 连续的大写字母只有首字母被保留，如 JSONObject -> jsonObject。
	// 一些特殊缩写或带数字可能无法转换成用户期望的结果，如 SSL、K8S、I18N 等，最好避开这些缩写。
	Camel ColumnNameConverter = ColumnNameConvertFunc(func(s string) string {
		n := len(s)
		if n == 0 {
			return ""
		}
		out := make([]byte, 0, n)
		capital := false
		for i := 0; i < n-2; i++ {
			if capital {
				out = append(out, toUpper(s[i]))
			} else {
				out = append(out, toLower(s[i]))
			}
			// out = append(out, toLower(s[i]))
			capital = isUpper(s[i+1]) && (isLower(s[i]) || isUpper(s[i]) && isLower(s[i+2]))
		}
		if n >= 2 {
			if capital {
				out = append(out, toUpper(s[n-2]))
			} else {
				out = append(out, toLower(s[n-2]))
			}
			capital = isUpper(s[n-1]) && isLower(s[n-2])
		}
		if capital {
			out = append(out, toUpper(s[n-1]))
		} else {
			out = append(out, toLower(s[n-1]))
		}
		return string(out)
	})
)

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func toLower(c byte) byte {
	if isUpper(c) {
		return c + ('a' - 'A')
	}
	return c
}

func toUpper(c byte) byte {
	if isLower(c) {
		return c - ('a' - 'A')
	}
	return c
}
