package log

import (
	"runtime"
	"strings"
	"time"

	"github.com/FlyingOnion/pkg/bytes"
)

type Formatter interface {
	FormatLevel(level Level) bytes.BufferWriter
	FormatTime(t time.Time) bytes.BufferWriter
	FormatCaller(callerSkip int) bytes.BufferWriter
	FormatMsg(msg string, keyValues ...any) bytes.BufferWriter
	MsgSplit() string
}

type TextFormatter struct{}

func (f *TextFormatter) FormatLevel(level Level) bytes.BufferWriter {
	return bytes.BufferWriteFunc(func(b *bytes.Buffer) {
		b.WriteByte('[').WriteString(strings.ToUpper(level.String())).WriteByte(']')
	})
}

func (f *TextFormatter) FormatTime(t time.Time) bytes.BufferWriter {
	return bytes.BufferWriteFunc(func(b *bytes.Buffer) {
		b.WriteTime(t, "2006-01-02 15:04:05.000")
	})
}

func (f *TextFormatter) FormatCaller(callerSkip int) bytes.BufferWriter {
	return bytes.BufferWriteFunc(func(b *bytes.Buffer) {
		pc, file, line, ok := runtime.Caller(callerSkip)
		b.If(ok).WriteString(file).WriteByte(':').WriteInt(line).
			WriteString(f.MsgSplit()).
			WriteString(runtime.FuncForPC(pc).Name()).
			Else().WriteString("#UnknownCaller#").EndIf()
	})
}

func (f *TextFormatter) FormatMsg(msg string, keyValues ...any) bytes.BufferWriter {
	return bytes.BufferWriteFunc(func(b *bytes.Buffer) {
		b.WriteString(msg).For(0, len(keyValues), 2, func(b *bytes.Buffer, i int) {
			b.WriteByte('\n').WriteString("  ").WriteAny(keyValues[i]).WriteString(": ").
				If(i == len(keyValues)-1).WriteString("#UnknownValue#").
				Else().WriteAny(keyValues[i+1]).
				EndIf()
		})
	})
}

func (f *TextFormatter) MsgSplit() string { return " " }
