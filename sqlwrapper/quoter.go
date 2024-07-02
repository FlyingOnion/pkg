package sqlwrapper

// Quoter 指定了加引号的方法
type Quoter interface {
	Quote(s string) string
}

type CustomQuoter struct {
	Prefix, Suffix string
}

func (q CustomQuoter) Quote(s string) string {
	b := make([]byte, 0, 3*len(s))
	b = append(b, q.Prefix...)
	i := 0
	for j := 0; j < len(s); {
		if s[j] != '.' {
			j++
			continue
		}
		b = append(b, s[i:j]...)
		b = append(b, q.Suffix...)
		b = append(b, '.')
		b = append(b, q.Prefix...)
		j++
		i = j
	}
	b = append(b, s[i:]...)
	b = append(b, q.Suffix...)
	return string(b)
}

var (
	// 不加引号，保持原样
	NoQuotes Quoter = CustomQuoter{}
	// 加双引号（postgresql、oracle 默认）
	DoubleQuotes Quoter = CustomQuoter{`"`, `"`}
	// 加单引号（很少用）
	SingleQuotes Quoter = CustomQuoter{"'", "'"}
	// 加方括号（sqlserver 默认）
	Brackets Quoter = CustomQuoter{"[", "]"}
	// 加反引号（mysql、sqlite 默认）
	Backticks Quoter = CustomQuoter{"`", "`"}
)
