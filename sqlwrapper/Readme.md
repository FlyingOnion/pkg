# sqlwrapper

SQL wrapper，自带一些有用的 ORM 功能。

[English](./Readme-EN.md)

**目录**

- [sqlwrapper](#sqlwrapper)
  - [数据库和驱动支持列表](#数据库和驱动支持列表)
  - [开始的第一步](#开始的第一步)
    - [1. 搭建本地环境](#1-搭建本地环境)
    - [2. 准备一个go程序](#2-准备一个go程序)
    - [3. 编译并运行](#3-编译并运行)
  - [结构体tag操作](#结构体tag操作)
  - [数据库操作](#数据库操作)
    - [Insert插入操作](#insert插入操作)
      - [创建一条新记录](#创建一条新记录)
      - [创建一条新记录（带主键）](#创建一条新记录带主键)
      - [指定插入的列](#指定插入的列)
      - [不忽略〇值字段](#不忽略〇值字段)
    - [Query查询操作](#query查询操作)
      - [查询指定列](#查询指定列)
      - [查询列时加上引号](#查询列时加上引号)
      - [手动指定表名](#手动指定表名)
      - [子查询](#子查询)
      - [Join表](#join表)
      - [Where 和值占位符](#where-和值占位符)
      - [GroupBy](#groupby)
      - [Having](#having)
      - [OrderBy, Limit, Offset](#orderby-limit-offset)
    - [Update更新操作](#update更新操作)
    - [Delete删除操作](#delete删除操作)
  - [扩展配置](#扩展配置)
    - [配置Dialect](#配置dialect)
    - [配置ValueConverter](#配置valueconverter)
    - [配置NULL值处理方式](#配置null值处理方式)
  - [完成进度](#完成进度)

## 数据库和驱动支持列表

|数据库|数据库版本|驱动|驱动版本（≥）|
|-|-|-|-|
|SQLite|3.35.5|github.com/mattn/go-sqlite3<br>modernc.org/sqlite|v1.14.13<br>v1.17.3|
|MySQL|8.0.27|github.com/go-sql-driver/mysql|v1.6.0|
|Postgresql|14.1|github.com/jackc/pgx/v4<br>github.com/lib/pq|v4.16.1<br>v1.10.2|
|SQLServer|2019-CU15-ubuntu-20.04|github.com/microsoft/go-mssqldb<br>github.com/denisenkom/go-mssqldb|v0.14.0<br>v0.12.3|
|Oracle|XE 18.4.0|github.com/mattn/go-oci8<br>github.com/sijms/go-ora<br>github.com/godror/godror|v0.1.1<br>testing<br>testing|


## 开始的第一步

### 1. 搭建本地环境

我们准备了即开即用的 MySQL、Postgresql、SQLServer 和 Oracle 的容器，只需要 git 和 docker-ce 即可搭建。

```bash
git clone --depth 1 --branch master https://github.com/FlyingOnion/local-db-docker.git
cd local-db-docker
docker compose up -d mysql

# or choose one of the following:

# docker compose up -d pg
# docker compose up -d oracle
# docker compose up -d mssql

# 用户名（登录名）：`orm`
# 密码：`ormO0I1pass`

# 这个密码也同样作为 root、postgres、sa、sys 特权用户的密码
```

### 2. 准备一个go程序

```go
package main

import (
  "fmt"

  _ "github.com/go-sql-driver/mysql"
  // 我们推荐用 . 来引入包，方便我们执行命令。
  . "github.com/FlyingOnion/pkg/sqlwrapper"
)

type Employee struct {
  ID       int64
  UserName string `db:"fullname"`
  Gender   int8
  Age      int
}

func (Employee) TableName() string { return "emp" }
func (Employee) PkColumn() string { return "id" }

func main() {
  db, _ := NewDatabase("mysql", "orm:ormO0I1pass@tcp(localhost:3306)/orm")
  var e Employee
  db.Query(&e, Where("id = ?", 1))
  fmt.Println(e)
  db.Close()
}
```

### 3. 编译并运行

假设你的 go 程序文件名字是 main.go，执行下面命令：

```shell
go mod init
go mod tidy
go build main.go -o query
./query
{1 Brian Mattal 1 20}
```

## 结构体tag操作

sqlwrapper 会检查结构体中名字为 `db` 的 tag。当 tag 为 `db:"-"` 时，这个字段会被忽略。其余情况下，sqlwrapper 会将 tag 中的值视为列名，并且**不会使用 ColumnNameConverter 转换**。

```go
type Employee struct {
  ID       int64
  UserName string `db:"fullname"`
  Gender   int8
  Age      int
  Ignored  int    `db:"-"` // 这个字段会被忽略
}
```

*使用 Oracle 的各位需要特别注意，写 tag 时请尽量使用全大写字母，比如 `db:"COLUMN_NAME"` 而不是 `db:"column_name"`。*

```go
type Employee struct {
  ID       int64
  UserName string `db:"fullname"` // 不太好的 tag，最好改成全大写
  Gender   int8
  Age      int
}

func (Employee) TableName() string { return "emp" } // 不太好的写法
func (Employee) PkColumn() string { return "id" } // 不太好的写法
```

```go
type Employee struct {
  ID       int64
  UserName string `db:"FULLNAME"` // 这样好一点
  Gender   int8
  Age      int
}

func (Employee) TableName() string { return "EMP" } // 这样好一点
func (Employee) PkColumn() string { return "ID" } // 这样好一点
```

## 数据库操作
### Insert插入操作

对 `*Database` 调用 Insert 方法即可插入记录。参数 e 需要实现 `IEntity` 接口指定表名和主键列名。

#### 创建一条新记录

```go
e := Employee{
  UserName: "NewEmployee",
  Gender: 1,
  Age: 30,
}
db.Insert(&e)
fmt.Println(e.ID) // <新记录的主键值>
```

#### 创建一条新记录（带主键）

`Employee` 的 `PkColumn` 返回值为 `"id"`，与 `ID` 字段对应，当 `e.ID` 不为〇值时，生成的 SQL 语句包含主键字段和值。

```go
e := Employee{
  ID: 20,
  UserName: "NewEmployee",
  Gender: 1,
  Age: 30,
}
db.Insert(&e)
```

#### 指定插入的列
用 `WithColumns` 可以指定插入的列。字符串传入数据库表的列名。
```go
e := Employee{
  UserName: "NewEmployee",
  Gender: 1,
  Age: 30,
}
db.Insert(&e, WithColumns("username", "age")) // Gender 字段的值会被忽略

var e1 Employee
db.Query(&e1, Where("id = ?", e.ID))
fmt.Println(e1.Gender) // 0
```

#### 不忽略〇值字段
结构体中的〇值字段默认会被忽略，即不会出现在 SQL 语句中，用 `IncludingZeros` 可以显式指定包含这些字段。

**适用于在数据库字段定义为 NOT NULL 但没有指定默认值的情况。**
```go
e := Employee{
  UserName: "NewEmployee",
  Gender: 0,
  Age: 0,
}
db.Insert(&e) // Gender、Age 的值会被忽略，不会出现在 SQL 语句中

db.Insert(&e, IncludingZeros()) //  SQL 语句中将显式指定 gender 和 age 的值
```

### Query查询操作

根据赋值目标变量不同，查询操作分为查询并赋值到结构体，和赋值到结构体切片，分别调用 `Query` 和 `QueryMultiple` 方法。

可用参数列举如下：

- 两者通用： `Select`, `From`, `Where`, `GroupBy`, `Having`, `OrderBy`, `Offset`
- 仅用于 `Query`：`RetrieveUnusedTo`
- 仅用于 `QueryMultiple`：`Limit` （`Query` 会强制 `limit 1`）

```go
var e Employee
found, err := db.Query(&e, Where("id = ?", 1))
fmt.Printf("%+v\n", e)
fmt.Println("found:", found)
fmt.Println("err:", err)
```

```go
// 可定义结构体切片，也可定义结构体指针切片
es := []Employee{}
// es := []*Employee{}
err := db.QueryMultiple(&es, Where("age > ?", 10))
for _, e := range es {
  fmt.Printf("%+v\n", e)
}
fmt.Println("err:", err)
```

#### 查询指定列
除了使用 `db:"-"` tag 忽略某个字段对应的列这个方法以外，还可以用 `Select` 手动指定查询哪些列。

```go
var e Employee
db.Query(&e,
  Select("id", "fullname"),
  Where("id = ?", 1),
)
fmt.Printf("%+v\n", e)
```

`*`、`distinct`、`count`、`sum`、`avg`、`min`、`max`、`as 别名` 等特殊字符串均直接写在字符串中即可。

```go
type Count struct {
  Count int64
}

var count Count
db.Query(&count,
  Select("count(*) as count"),
  From(Table("emp")),
  Where("age > ?", 20),
)
fmt.Println(count.Count)
```

#### 查询列时加上引号

适用于区分大小写或列名中包含数据库保留字的情况。
```go
var e Employee
quote := db.Quote
db.Query(&e,
  Select(quote("id"), quote("fullname")),
  Where("id = ?", 1),
)
fmt.Printf("%+v\n", e)
```

#### 手动指定表名

如果 Query 的参数 e 实现了 `IEntity` 接口，则默认使用 `e.TableName()` 作为表名。

需要手动指定表名时，可以用 `From(Table("foo"))`。

需要使用子查询时，可以用 `From(SubQuery(...))`，见 [Subquery](#subquery) 节。

```go
var e Employee
db.Query(&e,
  From(Table("emp")),
  Where("id = ?", 1),
)
fmt.Printf("%+v\n", e)
```

```go
es := []Employee{}
db.QueryMultiple(&es,
  Select("*"),
  From(
    Table("emp"),
    As("e")),
  Where("e.age > ?", 10),
)
for _, e := range es {
  fmt.Printf("%+v\n", e)
}
```

#### 子查询
用 `From(SubQuery(...), As("alias"))` 即可使用子查询。

`SubQuery` 的可选参数与 `QueryMultiple` 完全相同，你可以在里面再次开始 `Select` `From` `Where` `GroupBy` `Having` `OrderBy` `Limit` `Offset`。

通常情况下 `SubQuery` 的派生表都需要使用 `As` 指定别名，防止出现 `Every derived table must have its own alias` 报错。

```go
es := []Employee{}
db.QueryMultiple(&es,
  Select("*"),
  From(
    SubQuery(
      Select("*"),
      From(Table("emp")),
      Where("fullname like ?", "%Van%")),
    As("e")),
  Where("e.age > ?", 30)
)
for _, e := range es {
  fmt.Printf("%+v\n", e)
}
```

#### Join表
我们提供了 `InnerJoin`、`LeftJoin`、`RightJoin` 和 `FullJoin` 方法，可选参数均相同。你可以多次使用相同或不同的 Join 函数。

每个 Join 都需要用 `On` 或 `Using` 指定 join condition，比如：

`On("foo.bar_id = bar.id", "foo.baz_id = baz.id")`

`Using("bar_id", "baz_id")`

```go
es := []Employee{}
db.QueryMultiple(&es,
  From(
    Table("emp"),
    InnerJoin(
      Table("empn"),
      Using("fullname"),
    ),
    LeftJoin(...),
  ),
)
```

#### Where 和值占位符

常用数据库（见 [数据库和驱动支持列表](#数据库和驱动支持列表)）均支持使用 `?` 占位符。

当需要 `in` 或子查询时，占位符均使用 `?`，并传入一个 `ValueGroup` 或者 `SubQuery`。

```go
var e Employee
db.Query(&e, Where("id = ?", 1))

var es []Employee
db.QueryMultiple(&es, Where("id in ?", ValueGroup(3, 4, 5)))

// 注意：不是所有数据库都支持多列 in 语法。
// 目前只在 MySQL 和 Postgresql 支持。
db.QueryMultiple(&es, Where("(gender, age) in ?", ValueGroup(ValueGroup(1, 20), ValueGroup(0, 30))))

db.QueryMultiple(&es, Where("id in ?", SubQuery(
    Select("id"), From(Table("emp")), Where("age > ?", 20))))

db.QueryMultiple(&es, Where("age > all ?", SubQuery(
    Select("age"), From(Table("emp")), Where("gender = ?", 0))))

db.QueryMultiple(&es, Where("age > ?", SubQuery(
    Select("max(age)"), From(Table("emp")), Where("gender = ?", 0))))
```

#### GroupBy
直接传入列名，如 `GroupBy("foo", "bar", ...)`。


#### Having
直接传入条件字符串、占位符和值，如 `Having("sum(foo) > ?", 200)`。

#### OrderBy, Limit, Offset

直接传入条件即可。

```go
e := []Employee{}
db.QueryMultiple(&es,
  Where("age > ?", 25),
  OrderBy("fullname desc"),
  Limit(5),
  Offset(5),
)
```

### Update更新操作

文档待完善

```go
e := Employee{
  ID: 2,
  Name: "newName",
}
db.Update(&e)
```

### Delete删除操作

删除操作需要传入表名，以及条件。

不支持直接 `Delete` 一个 Entity。

```go
db.Delete("employee", Where("id = ?", 2))
```

## 扩展配置

sqlwrapper 支持扩展内部模块，比如 dialect 和 valueconverter。但通常情况下是不需要手动配置。

您需要对使用的数据库和驱动程序有充分的了解，否则生成的 SQL 语句可能会与数据库不兼容，或者在值域类型不匹配时出现错误。

### 配置Dialect

sqlwrapper 可以自定义 dialect，以支持其他数据库的配置。

Dialect 的原意是方言，指一些与数据库或驱动相关的配置。Dialect 中包括
- `ColumnNameConverter` 字段名和列名转换器
- `Placeholder` 占位符类型
- `Quoter` 引号（括号）类型

我们内置了 SQLite、MySQL、Postgresql、Oracle 和 SQLServer 相关的 dialect，以 driver 类型作为区分（如 `"mysql"`、`"pgx"`、`"oci8"`）。如果您的数据库使用的驱动类型不在内置中，则需要注册新的 dialect。您可以参考以下代码：

```go
dialect := CustomDialect{
  ColumnNameConverter: AsIs, // 不转换，列名即字段名
  PlaceHolder: QuestionMark, // 用问号作为占位符
  Quoter: NoQuotes,          // 不加引号或者括号
}
RegisterDialect("your_dialect", dialect)

db := NewDatabase("your_dialect", "your_database_address")
```

如果觉得麻烦，还有更简单的方法：直接使用 WithDialect 选项。这样做可以不写入全局的 dialect map，不会造成污染。

```go
dialect := CustomDialect{
  ColumnNameConverter: Snake,
  PlaceHolder: QuestionMark,
  Quoter: NoQuotes,
}
db := NewDatabase("your_dialect", "your_database_address",
  WithDialect(dialect),
)
```

`ColumnNameConverter`、`Placeholder`、`Quoter` 的可选项在下面列出。在下划线转化的解决方案上，我们参考了 [azer/snakecase](https://github.com/azer/snakecase)。在此表示感谢。

ColumnNameConverter：

```go
AsIs        // 列名 = 字段名
ToLower     // 列名 = 字段名转成全小写
ToUpper     // 列名 = 字段名转成全大写
Snake       // 列名 = 字段名小写下划线形式（默认，除 Oracle）
UpperSnake  // 列名 = 字段名大写下划线形式（Oracle 默认）
Camel       // 列名 = 字段名驼峰形式
```

PlaceHolder：

```go
QuestionMark // "?, ?, ?"
At_P_I       // "@p1, @p2, @p3"
Dollar_I     // "$1, $2, $3"
Colon_I      // ":1, :2, :3"
```

Quoter：

```go
NoQuotes     // 列名、表名不加引号或括号
DoubleQuotes // ""
SingleQuotes // ''
Brackets     // []
Backticks    // ``
```

### 配置ValueConverter

ValueConverter 的作用是将数据库中获取的值转换为目标变量类型的值。ORM 的默认转换器是 `Vcie`。`Vcie`在 `time.Time` 和 `string` 类型之间的转换时使用的默认时间格式是 `2006-01-02 15:04:05`。如果这个时间格式不符合您的要求，您需要包装一层后实现 `TimeFormat` 方法。

```go
type MyConverter struct {
  ValueConverter
}

func (MyConverter) TimeFormat() string {
  return time.RFC3339 // or other format
}

db := NewDatabase("your_dialect", "your_database_address",
  WithValueConverter(MyConverter{Vcie})
)
```

根据 `database/sql` 的规定，从数据库中获取的源值的类型必须转换为以下七种类型之一：`nil`, `int64`, `string`, `[]byte`, `bool`, `float64`, `time.Time`。这一步，数据库驱动已经替我们完成。因此 ORM 需要做的就是将这七种类型再转换到目标类型并对目标变量赋值。我们内置的 `Vcie` 已经可以处理大部分情况。如果您发现 Vcie 在转换时报错，可以提出 issue 或自己实现。

### 配置NULL值处理方式

根据经验，`NULL` 值在大部分情况下不太受欢迎，尤其是目标变量的类型不可以为 `nil` 的时候。`database/sql` 的处理方式简单粗暴，即报错返回 `error`。为了规避这种情况，目前有几种解决方法（以目标类型为 `string` 为例）：

- 用 `*string` 代替
- 用 `sql.NullString` 代替
- 建表时强制每个字段类型都设置 `not null default ""`
- 交给 ORM 处理

目前的 ORM 的处理方式也不尽相同：

- 用反射套一层指针后传入 Scan 方法（gorm v1）
- 先根据目标类型确定传入 Scan 方法的变量类型（如 `sql.NullString`），Scan 调用完再用不同的方法赋值到目标变量（gorm v2、xorm，我们称为“自定义 Scan”）
- 直接 switch type 逐个类型判断（zorm）

然而以上的方法，都缺少一个有时候很重要的步骤：让用户决定怎么做。

我们为用户提供了 3 种选择。在大部分情况下，保持默认的 `DoNothing` 即可。

- `DoNothing` （默认）：对目标变量不做任何事情，直接开始 Scan 下一个值
- `SetZero`：对目标变量赋〇值处理，如 `string` 类型则赋为空字符串，数字类型则赋为 0，`[]byte`、`interface{}` 和指针则赋为 `nil`
- `Continue`：交给 ValueConverter 处理（目前是直接报错返回）

```go
db := NewDatabase("your_dialect", "your_database_address",
  WithStrategyOnNull(DoNothing),
)
```

## 完成进度

- [x] 从结构体插入
- [ ] 从 `map[string]interface{}` 插入
- [x] 查询到指定结构体
- [x] 查询到指定结构体切片
- [ ] 查询到指定 map
- [x] 从带主键值的结构体更新
- [x] 删除
- [x] 提供不同的 `NULL` 值处理方式
- [x] 事务
- [ ] 支持不同的 logger
- [ ] 批量插入，提供 OnDuplicate（OnConflict）选项
- [ ] 批量更新，提供 OnDuplicate（OnConflict）选项
- [ ] 测试，测试，更多的测试
