package sqlwrapper

import "strconv"

type Placeholder interface {
	HoldPlace(index uint64) string
}

type PlaceholderFunc func(index uint64) string

func (fn PlaceholderFunc) HoldPlace(index uint64) string { return fn(index) }

var (
	// QuestionMark has the format (?, ?, ?), use in mysql, sqlite (default placeholder).
	//
	// 问号占位符 (?, ?, ?)
	QuestionMark Placeholder = PlaceholderFunc(func(index uint64) (s string) { return "?" })
	// At_P_I has the format (@p1, @p2, @p3), use in sqlserver.
	//
	// @p 加序号 (@p1, @p2, @p3)
	At_P_I Placeholder = PlaceholderFunc(func(index uint64) (s string) { return "@p" + strconv.FormatUint(index, 10) })
	// Dollar_I has the format ($1, $2, $3), use in postgresql, kingbase.
	//
	// 美元符号加序号 ($1, $2, $3)
	Dollar_I Placeholder = PlaceholderFunc(func(index uint64) (s string) { return "$" + strconv.FormatUint(index, 10) })
	// Colon_I has the format (:1, :2, :3), use in oracle, shentong.
	//
	// 冒号加序号 (:1, :2, :3)
	Colon_I Placeholder = PlaceholderFunc(func(index uint64) (s string) { return ":" + strconv.FormatUint(index, 10) })
)
