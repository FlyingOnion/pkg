# References
关于各常用数据库的一些参考资料。

- [SQLite](#sqlite)
  - [获取受SQL语句影响的行数](#获取受sql语句影响的行数)
- [MySQL](#mysql)
  - [当全量搜索时MySQL的行为](#当全量搜索时mysql的行为)
  - [单独指定offset，但不指定limit](#单独指定offset但不指定limit)
- [SQLServer](#sqlserver)
  - [@@ROWCOUNT官方文档](#rowcount官方文档)
  - [reliability of rowcount after update](#reliability-of-rowcount-after-update)
  - [scope of rowcount](#scope-of-rowcount)
  - [@@ROWCOUNT的线程安全](#rowcount的线程安全)
  - [GO命令](#go命令)
  - [varchar和nvarchar](#varchar和nvarchar)
- [Postgresql](#postgresql)
  - [将returning的结果作为表用于后续的SQL操作](#将returning的结果作为表用于后续的sql操作)
- [Oracle](#oracle)
  - [斜杠和分号](#斜杠和分号)
  - [CREATE TRIGGER SQL脚本无法执行](#create-trigger-sql脚本无法执行)
  - [Character Set And NChar Character Set](#character-set-and-nchar-character-set)
- [各库Query方法对分号的支持情况](#各库query方法对分号的支持情况)
- [各库对LastInsertId和RowsAffected的支持情况](#各库对lastinsertid和rowsaffected的支持情况)
- [Limit Offset和Offset Fetch Next](#limit-offset和offset-fetch-next)
- [以query语句返回last\_insert\_id和rows\_affected](#以query语句返回last_insert_id和rows_affected)
  - [SQLite](#sqlite-1)
  - [MySQL](#mysql-1)
  - [Postgresql](#postgresql-1)
- [database/sql convertAssign func](#databasesql-convertassign-func)

## SQLite

### 获取受SQL语句影响的行数

changes() 方法：获取最近一次结束的 statement 影响的行数。

total_changes() 方法：获取本次连接开始，截至当前时间中发生的行数变化总数。

原值更新也会算入变化。

https://stackoverflow.com/questions/6551903/getting-the-number-of-affected-rows-for-a-sqlite-statement-using-the-c-api

https://stackoverflow.com/questions/6551903/getting-the-number-of-affected-rows-for-a-sqlite-statement-using-the-c-api

## MySQL

### 当全量搜索时MySQL的行为

https://github.com/go-sql-driver/mysql/issues/407

https://github.com/go-sql-driver/mysql/issues/366

问题出现在 Query 测试时的一次 `unsupported conversion from type []uint8 into type int64` 错误。此时我们打印的日志显示每个字段都是 `[]byte` 类型。但如果加上 `Where` 条件搜索，则返回的类型是变量实际类型。

根据 julienschmidt 的回答，全部返回 `[]byte` 是 MySQL 本身的行为。意思是，驱动解析 MySQL 响应得到的结果就是“每一列数据全都是 `[]byte` 类型”。推测 MySQL 返回大量数据时，这样做可以减少服务器的性能开销。

根据我的测试结果，目前这种情况仅在全量搜索时出现。使用 `where 1 = 1` 这样的 trick 结果也一样。这种情况在 `database/sql` `convertAssignRows` 方法中，需要走到最后一步，即将源值转为 `string` 类型后再转为目标类型才能完成转换。在我们的 Vcie 中，源类型为 `[]byte` 时走的是 `ConvertBytes` 方法，因此我们在最后一步也加上了 `ConvertString(string(src))`。

在加上 `ConvertString(string(src))` 转换之前，从 `[]byte` 到 `int64` 这种转换会报错。以下为实验结果：

```sql
-- 出现错误（id 列为 []byte 类型）
select * from emp
select 1 as id from emp
select id from emp
select * from emp order by fullname
select * from emp where 1 = 1

-- 不出现错误（id 列为 int64 类型）
select * from emp where id = 2
select * from emp limit 2
```

**总结：全量搜索时 MySQL 会摆烂，全部返回字节数组交给应用自己转换。此时 MySQL 效率最高，但会加重应用侧（我们）进行类型转换的性能开销。**

### 单独指定offset，但不指定limit

https://stackoverflow.com/questions/255517/mysql-offset-infinite-rows

https://dev.mysql.com/doc/refman/8.0/en/select.html

指定偏移量，选出偏移量位置的记录到最后一条记录。

官方解决方案是使用 18446744073709551615 （2^64-1），但被很多人吐槽 awful。

MariaDB 没有对该“特性”进行改进。

## SQLServer

### @@ROWCOUNT官方文档

https://docs.microsoft.com/zh-cn/sql/t-sql/functions/rowcount-transact-sql?view=sql-server-ver15

### reliability of rowcount after update

假如 update 将某行按原值更新时 @@ROWCOUNT 的值的情况，并与 MySQL ROWCOUNT 进行比较。结果是 SQLServer 不管是不是原值更新，只要 where 匹配到的，都算在 @@ROWCOUNT 中。而 MySQL 的 ROWCOUNT 只计算更新了值的行数。

https://stackoverflow.com/questions/23954516/is-rowcount-after-update-reliably-a-measure-of-matching-rows

### scope of rowcount

触发器不会影响 @@ROWCOUNT 的值。

https://stackoverflow.com/questions/11834980/scope-of-rowcount

### @@ROWCOUNT的线程安全

@@ROWCOUNT 是线程安全（scope safe 和 connection safe）的，总会返回当前连接的上一条语句影响的行数。但调用后值就会置为 0。所以如果需要多次使用这个值，就需要将这个值保存到临时变量里。

https://stackoverflow.com/questions/8960510/sql-server-is-using-rowcount-safe-in-multithreaded-applications

### GO命令

经检验，`CREATE DATABASE` 后必须加上 `GO` 命令，否则后续无法在该数据库中建表。

其他情况似乎可以多个语句写完再加上一块执行。以防万一，建议在 `CREATE LOGIN`、`CREATE USER` 等语句后执行一遍 `GO` 命令。

### varchar和nvarchar

nvarchar 支持中文和 emoji 等特殊字符。

https://www.cnblogs.com/glory0727/p/10061337.html

SQL 脚本中，nvarchar 字符串值必须使用 `N'xxx'` 格式显式指定。

`microsoft/go-mssqldb` 库中的 string 的默认类型为 nvarchar。参见

https://github.com/microsoft/go-mssqldb/blob/2d408c3ae25a3af94a54322903b14c854e98b18c/mssql.go#L979

https://github.com/microsoft/go-mssqldb/blob/2d408c3ae25a3af94a54322903b14c854e98b18c/mssql.go#L910

考虑到语言兼容性，大部分情况下推荐使用 nvarchar。但根据 [issue 724](https://github.com/denisenkom/go-mssqldb/issues/724) 所述，记录数量多时 nvarchar 列索引性能会出现明显降低。

因此 **当某列的值中只包含英文、数字，且该列有索引且该列的索引性能较为重要时，可以考虑将该列设为 varchar。**

## Postgresql

### 将returning的结果作为表用于后续的SQL操作

× 以下方法无法通过编译：
```sql
insert into ... values (...) returning id limit 1
select * from (insert into ... values (...) returning id)
```

✔ 以下方法可以通过编译（且是原子性的，select 语句语法出错时不会 insert）：
```sql
with exec_result as (insert into ... returning id)
select * from exec_result where ... order by ... limit 1
```

## Oracle

### 斜杠和分号

`;` 是执行语句必须的。

`/` 是执行语句块必须的。

https://www.cnblogs.com/songzhenghe/p/4582319.html

### CREATE TRIGGER SQL脚本无法执行

CREATE TRIGGER 是语句块，斜杠 `/` 不能少。

```sql
CREATE TRIGGER ... WHEN ...
DECLARE
  -- some variables
BEGIN
  -- do something
END;
/
```

### Character Set And NChar Character Set

根据该[机翻文档](https://www.askmac.cn/archives/%E5%A6%82%E4%BD%95%E9%80%89%E6%8B%A9%E6%88%96%E6%9B%B4%E6%94%B9%E6%95%B0%E6%8D%AE%E5%BA%93%E5%AD%97%E7%AC%A6%E9%9B%86-nls_characterset-doc-id-1525394-1.html)的描述。Oracle 一般建议使用 NLS_CHARACTERSET、CHAR,、VARCHAR2、LONG 和 CLOB 这样的数据类型,而不是 N- 类型的数据类型。

根据我的实验，大多数情况下使用 AL32UTF8 格式的 Charset 可以存储大多数 Unicode 字符（包括中文）。但 emoji 比较特殊。以下为实验结果：

|表列定义|插入值|Navicat 显示结果|
|-|-|-|
|varchar2(100)|`'Chris Weibel😄'`|`Chris Weibel����`|
|nvarchar2(100)|`'Chris Weibel😄'`|`Chris Weibel����`|
|nvarchar2(100)|`N'Chris Weibel😄'`|`Chris Weibel����`|
|varchar2(100 char)|`'Chris Weibel😄'`|`Chris Weibel����`|

## 各库Query方法对分号的支持情况

此处指该驱动能否支持调用 db.Query 时将两个或多个 SQL 语句写在一个字符串中并使用分号分隔。

**注：两个语句一起执行的情况仅在极少情况下出现，通常情况下推荐分开调用或使用事务。**

|库|版本|支持情况|
|-|-|-|
|mattn/go-sqlite3|v1.14.13|仅最后一条语句执行（参见 [issue 933](https://github.com/mattn/go-sqlite3/issues/933#issuecomment-998388064)）|
|modernc.org/sqlite|v1.17.3|未测试|
|go-sql-driver/mysql|v1.6.0|仅支持多个 exec 类型语句，不支持 exec 后接 query 语句|
|microsoft/go-mssqldb|v0.14.0|未测试|
|jackc/pgx/v4|v4.16.1|未测试|
|lib/pq|v1.10.6|未测试|
|mattn/go-oci8|v0.1.1|未测试|

## 各库对LastInsertId和RowsAffected的支持情况

|库|支持情况|原理|
|-|-|-|
|mattn/go-sqlite3|支持|调用 c 库中 sqlite3_last_insert_rowid 和 sqlite3_changes|
|modernc.org/sqlite|支持|调用 ccgo 方法（由代码生成器生成）|
|go-sql-driver/mysql|支持|解析 response|
|microsoft/go-mssqldb<br/>denisenkom/go-mssqldb|只支持 RowsAffected|解析 response|
|jackc/pgx/v4|只支持 RowsAffected|解析 response|
|lib/pq|只支持 RowsAffected|解析 response|
|mattn/go-oci8|支持|调用 c 库中方法（没仔细找）|

## Limit Offset和Offset Fetch Next
SQLite、MySQL 只支持 limit offset，但 Offset 是 Limit 的参数。
```sql
[ LIMIT count [ OFFSET count ] ]
```

https://sqlite.org/lang_select.html

https://dev.mysql.com/doc/refman/8.0/en/select.html

---

Postgresql 两种都支持，且语法较宽松，可以单独使用任意一个。
```sql
[ LIMIT { count | ALL } ]
[ OFFSET start [ ROW | ROWS ] ]
[ FETCH { FIRST | NEXT } [ count ] { ROW | ROWS } { ONLY | WITH TIES } ]
```

https://www.postgresql.org/docs/14/sql-select.html#SQL-LIMIT

---

SQLServer（2012 及以后版本）, Oracle（12c 及以后版本）只支持 offset + fetch next。SQLServer 的语法要求比较严格，offset 和 fetch 均为 order by 的参数。

https://docs.oracle.com/en/database/oracle/oracle-database/19/sqlrf/SELECT.html

https://learn.microsoft.com/en-us/sql/t-sql/queries/select-order-by-clause-transact-sql?view=sql-server-2017#syntax

```sql
--- Oracle
[ ORDER BY column ]
[ OFFSET count ROWS ]
[ FETCH { FIRST | NEXT } count ROWS { ONLY | WITH TIES } ]
--- SQLServer
[ ORDER BY column [ OFFSET count ROWS [ FETCH NEXT count ROWS ONLY ] ]
```


## 以query语句返回last_insert_id和rows_affected

### SQLite

```sql
-- insert, update
insert into {table_name} ...; select last_insert_rowid(), changes();
update {table_name} set ...; select 0, changes();
```

### MySQL

```sql
-- mysql workbench 运行正常，但 adminer 运行时 row_count 返回 -1。原因不详。

```

### Postgresql

```sql
-- insert
with exec_result as (insert into ... returning id) select max(id) as last_insert_id, count(id) as affected_rows from exec_result;

-- update
with exec_result as (update {table_name} set ... returning id) select 0 as last_insert_id, count(id) as affected_rows from exec_result
```

## database/sql convertAssign func

Part of conversion methods.

* S: first switch src type, then switch dest type
* D: first switch dest type, then switch or reflect src type
* =: equals to directly set `dest = src`
* R: directly reflection set (same kind and convertible), or src -> string -> dest (otherwise)
* -: not supported, return error

<table>
<thead>
<tr>
<th>dest \ src</th>
<th><code>int64</code></th>
<th><code>bool</code></th>
<th><code>string</code></th>
<th><code>float64</code></th>
<th><code>time.Time</code></th>
<th><code>[]byte</code></th>
<th><code>nil</code></th>
</tr>
</thead>
<tbody>
<tr>
<td><code>*string</code></td><td bgcolor=#ffc0cb>D</td><td bgcolor=#ffc0cb>D</td><td bgcolor=#66ffc0>S</td><td bgcolor=#ffc0cb>D</td><td bgcolor=#ffff77>S</td><td bgcolor=#77ccff>S</td><td>-</td>
</tr>
<tr>
<td><code>*[]byte</code><br><code>*RawBytes</code></td><td bgcolor=#bbffff>D</td><td bgcolor=#bbffff>D</td><td bgcolor=#66ffc0>S</td><td bgcolor=#bbffff>D</td><td bgcolor=#ffff77>S</td><td bgcolor=#77ccff>S</td><td bgcolor=#bb6600>S</td>
</tr>
<tr>
<td><code>*interface{}</code></td><td bgcolor=#c0c0cc style="text-align: center;" colspan=5>=</td><td bgcolor=#77ccff>S</td><td bgcolor=#bb6600>S</td>
</tr>
<tr>
<td><code>*time.Time</code></td><td>-</td><td>-</td><td>-</td><td>-</td><td bgcolor=#ffff77>S</td><td>-</td><td>-</td>
</tr>
<tr>
<td><code>*bool</code></td><td bgcolor=#ccaaff>D</td><td bgcolor=#ccaaff>D</td><td bgcolor=#ccaaff>D</td><td>-</td><td>-</td><td bgcolor=#ccaaff>D</td><td>-</td>
</tr>
<tr>
<td><code>*ints<br>*uints<br>*float32(64)</code></td><td bgcolor=#999999 style="text-align: center; color: white" colspan=6><strong>R</strong></td><td>-</td>
</tr>
</tbody>
</table>