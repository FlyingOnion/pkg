package sqlwrapper

type IEntity interface {
	// TableName 获取表名称，与 gorm model 通用。
	//
	// 最好先定义表名的常量字符串，调用 TableName 时直接将该常量返回。
	TableName() string

	// PkColumn 获取数据库表的主键字段名称。不支持复合主键。
	PkColumn() string
}
