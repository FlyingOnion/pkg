package sqlwrapper

import (
	"reflect"
)

var metadata = map[reflect.Type]*structMeta{}

// structMeta 是结构体的元数据，只与该结构体的结构有关，与变量无关。
//
// 使用 structMeta 可以防止每次都花费大量时间使用反射获取类型信息。
type structMeta struct {
	// 结构体字段数。
	nFields int

	// 主键对应的字段在结构体中的位置，从 0 开始。在调用 RegisterType 分析结构体时获取。
	//
	// 结构体字段（如果有 tag 的话）的 tag 名或（如果没有 tag 的话）经过转换的字段名如果与 entity 的 PkColumn 结果一致，就认定该字段为主键字段，并将该字段的 Index 赋值给 pkStructIndex。
	//
	// -1 代表结构体中没有一个字段能对应主键的列。此时若进行 insert 操作则无法返回新记录的 ID 并赋值给 entity。
	pkStructIndex int

	// 主键的类型（只能是 int 家族、uint 家族或 string，且不能是指针）。
	pkType reflect.Type

	// columns 记录了所有 tag 不为 "-" 的字段对应的列的名字。长度不大于 nFields。
	columns []string

	// columnFieldMap 以结构体字段的列名为 Key，元数据为 Value。
	//
	// 当字段的 tag 为 "-" 时，columnFieldMap 不记录该字段。
	//  len(columnFieldMap) == len(columns)
	columnFieldMap map[string]*fieldMeta
}

// fieldMeta 是结构体中字段的元数据，只与该字段在结构体中的位置（index）和字段类型有关，与该字段的值无关。
type fieldMeta struct {
	index  int
	column string
}

// 注册类型信息，entity 必须是结构体或结构体指针。
//
// 由于程序无法在运行时改变一个结构体的结构，所有类型信息都可以存在全局变量 metadata 中。
func (db *Database) RegisterType(entity interface{}) (*structMeta, error) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, ErrElemNotStruct
	}
	// 如果已经注册，则直接返回
	if sm, found := metadata[t]; found {
		return sm, nil
	}

	// 初始化
	nFields := t.NumField()
	cols := make([]string, 0, nFields)
	cfmap := make(map[string]*fieldMeta, nFields)

	pkColumn := ""
	if iEntity, ok := entity.(IEntity); ok {
		pkColumn = iEntity.PkColumn()
	}
	pkStructIndex := -1
	pkType := reflect.Type(nil)

	// 遍历每个字段，解析 tag、构建字段名、判断字段类型以确定该字段 Value 的 ScanValue。
	for i := 0; i < nFields; i++ {
		field := t.Field(i)
		// 跳过匿名字段
		if field.Anonymous {
			continue
		}
		colName := field.Tag.Get("db")
		// 跳过 db:"-" 的字段
		if colName == "-" {
			continue
		}
		if len(colName) == 0 {
			// 如果 tag db 是空字符串，则使用 dialect 的转换方法将字段名转换为数据库列名。
			colName = db.dialect.Convert(t.Field(i).Name)
		}

		// 主键字段通常会写在结构体第一位，因此第一次循环就会触发。
		// 第二次循环之后用 pkStructIndex == -1 作为短路条件效率比较高。
		if pkStructIndex == -1 && colName == pkColumn {
			// 限制主键字段类型。
			switch field.Type.Kind() {
			case reflect.Int,
				reflect.Int8,
				reflect.Int16,
				reflect.Int32,
				reflect.Int64,
				reflect.Uint,
				reflect.Uint8,
				reflect.Uint16,
				reflect.Uint32,
				reflect.Uint64,
				reflect.String:
				// pass
			default:
				return nil, ErrInvalidPKType
			}
			pkStructIndex, pkType = i, field.Type
		}

		cols = append(cols, colName)
		fm := &fieldMeta{
			index:  i,
			column: colName,
		}
		cfmap[colName] = fm
	}
	// 构建结构体的元数据，添加到 metadata
	sm := &structMeta{
		nFields:        nFields,
		pkStructIndex:  pkStructIndex,
		pkType:         pkType,
		columns:        cols,
		columnFieldMap: cfmap,
	}
	metadata[t] = sm
	return sm, nil
}
