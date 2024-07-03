# pkg
This project `github.com/FlyingOnion/pkg` provides some interesting go packages.

- `bytes` provides a `Buffer` that could be used as a substitution for official `bytes.Buffer`, with a lot of chaining `Write` methods and `If`, `ElseIf`, `Else`, `For` and `Break`;
- `log` is a **radical** logging package without any `f` methods; instead, it improves the efficiency of logging;
- `context` provides context with multiple parents, `ChainingContext` and `ResetContext`;
- `sqlwrapper` is a sql wrapper with some useful ORM features. See [Readme-CN](sqlwrapper/Readme.md) or [Readme-EN](sqlwrapper/Readme-EN.md) for more details;

## bytes

TL;DR

```go
import "github.com/FlyingOnion/pkg/bytes"

func getAddr(useHttps bool, host string, port int) string {
    var b bytes.Buffer
    return b.
        If(useHttps).WriteString("https://").Else().WriteString("http://").End().
        WriteString(host).
        If(port != 80).WriteString(":").WriteInt(port).End().
        String()
}
```

## context

ChainingContext is a powerful tool to control the time spent on each task.

```go
ctx, cancel := context.WithFractions(parent, 0.3, 0.2, 0.2)
defer cancel()
doTask1(ctx.Next()) // use 30% of time to do task1
doTask2(ctx.Next()) // use 20% of time to do task2
doTask3(ctx.Next()) // use 20% of time to do task3
doTask4(ctx.Next()) // use a sentinel context to do task4
```

ResetContext includes a timer and it could be reset. If the time is up, the context will be canceled.

```go
func process(conn net.Conn) {
    ctx, cancel := context.WithReset(5 * time.Second)
    defer cancel()
    go func() {
        <-ctx.Done()
        conn.Close()
    }()
    var buf bytes.Buffer
    for {
        n, err := io.Copy(&buf, conn)
        if err != nil {
            break
        }
        ctx.Reset(5 * time.Second)
        // do something
    }
}

// server
l, _ := net.Listen("tcp", ":8080")
defer l.Close()

for {
    conn, _ := l.Accept()
    process(conn)
}
```