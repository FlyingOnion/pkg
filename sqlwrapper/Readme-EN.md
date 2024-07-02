# sqlwrapper

A SQL wrapper with some useful ORM features.

- [sqlwrapper](#sqlwrapper)
  - [Supported Drivers](#supported-drivers)
  - [Get Started](#get-started)
    - [1. Prepare Environment](#1-prepare-environment)
    - [2. Write a Program](#2-write-a-program)
    - [3. Build and Run](#3-build-and-run)
  - [Struct Tags](#struct-tags)
  - [Database Operations](#database-operations)
    - [Insert](#insert)
      - [Without Primary Key Value](#without-primary-key-value)
      - [With Primary Key Value](#with-primary-key-value)
      - [With Specific Columns](#with-specific-columns)
      - [With Zero Values](#with-zero-values)
    - [Query](#query)
      - [Select Specific Columns](#select-specific-columns)
      - [Quoter](#quoter)
      - [Table Name](#table-name)
      - [SubQuery](#subquery)
      - [Table Joining](#table-joining)
      - [Where and Column Placeholder](#where-and-column-placeholder)
      - [GroupBy](#groupby)
      - [Having](#having)
      - [OrderBy, Limit, Offset](#orderby-limit-offset)
    - [Update](#update)
    - [Delete](#delete)
  - [Extension](#extension)
    - [Dialect Extension](#dialect-extension)
    - [ValueConverter Extension](#valueconverter-extension)
    - [NULL Value Handling](#null-value-handling)
  - [Process](#process)

## Supported Drivers

|Database|Version|Drivers|Version(≥)|
|-|-|-|-|
|SQLite|3.35.5|github.com/mattn/go-sqlite3<br>modernc.org/sqlite|v1.14.13<br>v1.17.3|
|MySQL|8.0.27|github.com/go-sql-driver/mysql|v1.6.0|
|Postgresql|14.1|github.com/jackc/pgx/v4<br>github.com/lib/pq|v4.16.1<br>v1.10.2|
|SQLServer|2019-CU15-ubuntu-20.04|github.com/microsoft/go-mssqldb<br>github.com/denisenkom/go-mssqldb|v0.14.0<br>v0.12.3|
|Oracle|XE 18.4.0|github.com/mattn/go-oci8<br>github.com/sijms/go-ora<br>github.com/godror/godror|v0.1.1<br>testing<br>testing|

## Get Started

### 1. Prepare Environment

Use Docker to run a local database.

```bash
git clone --depth 1 --branch master https://github.com/FlyingOnion/local-db-docker.git
cd local-db-docker
docker compose up -d mysql

# or choose one of the following:

# docker compose up -d pg
# docker compose up -d oracle
# docker compose up -d mssql
```

### 2. Write a Program

```go
package main

import (
  "fmt"

  _ "github.com/go-sql-driver/mysql"
  // we suggest that you use local import function calls
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

### 3. Build and Run

```shell
go mod init
go mod tidy
go build main.go -o query
./query
{1 Brian Mattal 1 20}
```

## Struct Tags

Sqlwrapper will check tags of struct fields. If the tag of a field is `db:"-"`, this field will be ignored. 

Otherwise, **the tag will be used as the column name DIRECTLY**, and `ColumnNameConverter` will not be applied.

```go
type Employee struct {
  ID       int64
  UserName string `db:"fullname"`
  Gender   int8
  Age      int
  Ignored  int    `db:"-"` // this field will be ignored
}
```

*When using Oracle, please use uppercase tags like `db:"COLUMN_NAME"` to avoid case-sensitive problems.*

```go
type Employee struct {
  ID       int64
  UserName string `db:"fullname"` // NOT SO GOOD
  Gender   int8
  Age      int
}

func (Employee) TableName() string { return "emp" } // NOT SO GOOD
func (Employee) PkColumn() string { return "id" } // NOT SO GOOD
```

```go
type Employee struct {
  ID       int64
  UserName string `db:"FULLNAME"` // IT'S BETTER
  Gender   int8
  Age      int
}

func (Employee) TableName() string { return "EMP" } // IT'S BETTER
func (Employee) PkColumn() string { return "ID" } // IT'S BETTER
```

## Database Operations
### Insert

Calling `Insert` will insert a new record into the database. Parameter `e` must be a pointer to a struct and implements `TableName` and `PkColumn` methods.

#### Without Primary Key Value

```go
e := Employee{
  UserName: "NewEmployee",
  Gender: 1,
  Age: 30,
}
db.Insert(&e)
fmt.Println(e.ID) // <id of new record>
```

#### With Primary Key Value

`PkColumn` of `Employee` is `"id"`, which is mapped by field `ID`. When `e.ID` is not zero, `Insert` will insert a new record with `id` equals to `e.ID`.

```go
e := Employee{
  ID: 20,
  UserName: "NewEmployee",
  Gender: 1,
  Age: 30,
}
db.Insert(&e)
```

#### With Specific Columns

Use `WithColumns` option to specify which columns to insert.

```go
e := Employee{
  UserName: "NewEmployee",
  Gender: 1,
  Age: 30,
}
// value of Gender field will be ignored because it's not in the list
db.Insert(&e, WithColumns("username", "age"))

var e1 Employee
db.Query(&e1, Where("id = ?", e.ID))
fmt.Println(e1.Gender) // 0
```

#### With Zero Values

Use `IncludingZeros` option to insert zero values, or they will be ignored by default.

```go
e := Employee{
  UserName: "NewEmployee",
  Gender: 0,
  Age: 0,
}

db.Insert(&e) // Gender, Age will not appear in sql statement.

db.Insert(&e, IncludingZeros()) // Gender, Age will appear in sql statement.
```

### Query

To query data from database, use `Query` or `QueryMultiple` method, depending on the target is a single entity struct or an entity slice.

Options of `Query` and `QueryMultiple`:

- Common: `Select`, `From`, `Where`, `GroupBy`, `Having`, `OrderBy`, `Offset`
- `Query` only：`RetrieveUnusedTo(map[string]interface{})`
- `QueryMultiple` only：`Limit` (`Query` will automatically set `Limit` to 1)

```go
var e Employee
found, err := db.Query(&e, Where("id = ?", 1))
fmt.Printf("%+v\n", e)
fmt.Println("found:", found)
fmt.Println("err:", err)
```

```go
// Both struct slice and struct pointer slice are supported
es := []Employee{}
// es := []*Employee{}
db.QueryMultiple(&es, Where("age > ?", 10))
for _, e := range es {
  fmt.Printf("%+v\n", e)
}
```

#### Select Specific Columns

Fields with tag `db:"-"` will be automatically ignored.

In addition, you can use `Select` option to specify columns.

```go
var e Employee
db.Query(&e,
  Select("id", "fullname"),
  Where("id = ?", 1),
)
fmt.Printf("%+v\n", e)
```

Special columns like `*`、`distinct`、`count`、`sum`、`avg`、`min`、`max`、`as <alias>` can be also used.

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

#### Quoter

Quoter is used to quote the tables and columns. It is suitable when the database is case-sensitive or when the table or column name contains reserved keywords.

```go
var e Employee
quote := db.Quote
db.Query(&e,
  Select(quote("id"), quote("fullname")),
  Where("id = ?", 1),
)
fmt.Printf("%+v\n", e)
```

#### Table Name

If entity `e` implements `IEntity` interface, then the table name will be automatically set to the value returned by `e.TableName()`.

Otherwise, use `From(Table("foo"))` to set the table name, or `From(Subquery(...))` if you want a derived table. See [Subquery](#subquery) for more details.

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

#### SubQuery

Use `From(SubQuery(...), As("alias"))` option to create a subquery.

In common cases, `As()` option should always be used to avoid sql syntax errors like `Every derived table must have its own alias`.

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

#### Table Joining

`InnerJoin`, `LeftJoin`, `RightJoin`, `FullJoin` are supported. You can use them multiple times if needed.

For each join, you need to provide an `On` or `Using` JoinCondition, like:

`On("foo.bar_id = bar.id", "foo.baz_id = baz.id")`

`Using("bar_id", "baz_id")`

```go
es := []Employee{}
db.QueryMultiple(&es,
  From(
    Table("emp"), // or SubQuery(...),
    InnerJoin(
      Table("empn"), // or SubQuery(...),
      Using("fullname"),
    ),
    LeftJoin(...),
  ),
)
```

#### Where and Column Placeholder

We have already supported `?` placeholder in common databases.

```go
var e Employee
db.Query(&e, Where("id = ?", 1))

var es []Employee
db.QueryMultiple(&es, Where("id in ?", ValueGroup(3, 4, 5)))

// Not all databases support "multiple column in".
// Currently only MySQL and Postgresql support it.
db.QueryMultiple(&es, Where("(gender, age) in ?", ValueGroup(ValueGroup(1, 20), ValueGroup(0, 30))))

db.QueryMultiple(&es, Where("id in ?", SubQuery(
    Select("id"), From(Table("emp")), Where("age > ?", 20))))

db.QueryMultiple(&es, Where("age > all ?", SubQuery(
    Select("age"), From(Table("emp")), Where("gender = ?", 0))))

db.QueryMultiple(&es, Where("age > ?", SubQuery(
    Select("max(age)"), From(Table("emp")), Where("gender = ?", 0))))
```

#### GroupBy
Pass column names directly, like `GroupBy("foo", "bar", ...)`.

#### Having
Pass condition `Having("sum(foo) > ?", 200)`

#### OrderBy, Limit, Offset

```go
e := []Employee{}
db.QueryMultiple(&es,
  Where("age > ?", 25),
  OrderBy("fullname desc"),
  Limit(5),
  Offset(5),
)
```

### Update

TODO: Add more details

```go
e := Employee{
  ID: 2,
  Name: "newName",
}
db.Update(&e)
```

### Delete

To delete a record, use `Delete` method with table name and condition. 

**We do not support `Delete` method with entity struct.**

```go
db.Delete("employee", Where("id = ?", 2))
```

## Extension



### Dialect Extension

Sqlwrapper supports custom dialects to support more databases.

In a dialect, you should provide the following methods:

- `ColumnNameConverter`: To convert struct field names to table column names.
- `Placeholder`: To specify the placeholder of values.
- `Quoter`: To specify the quoter.

`SQLite`, `MySQL`, `Postgresql` and `SQLServer`, and `Oracle` are built-in dialects. Their driver names are their identifiers, such as `mysql`, `pgx`, `oci8`.

If you need a custom dialect, see codes below as a reference.

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

Available ColumnNameConverters:

```
AsIs
ToLower
ToUpper
Snake (default except Oracle)
UpperSnake (Oracle default)
Camel
```

Available PlaceHolders:

```go
QuestionMark // "?, ?, ?"
At_P_I       // "@p1, @p2, @p3"
Dollar_I     // "$1, $2, $3"
Colon_I      // ":1, :2, :3"
```

Available Quoters:

```go
NoQuotes // default
DoubleQuotes // ""
SingleQuotes // ''
Brackets     // []
Backticks    // ``
```

We referenced [azer/snakecase](https://github.com/azer/snakecase) to implement `Snake` and `UpperSnake` ColumnNameConverters. I appreciate the author's work.

### ValueConverter Extension

TODO: Add more details

### NULL Value Handling

TODO: Add more details

## Process

- [x] Insert from struct entity
- [ ] Insert from `map[string]interface{}`
- [x] Query to entity
- [x] Query to slice of entity
- [ ] Query to map
- [x] Update
- [x] Delete
- [ ] Provide different `NULL` value handling
- [x] Transaction
- [ ] Support common loggers
- [ ] Batch Insert
- [ ] Batch Update
- [ ] Test, test, more test
