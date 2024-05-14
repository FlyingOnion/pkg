package log

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/FlyingOnion/pkg/bytes"
)

func AlarmPanic(msg string)               { panic(msg) }
func AlarmFatal(msg string, exitCode int) { os.Exit(exitCode) }

type Logger struct {
	level      Level
	callerSkip int
	// fields is a sequence of fields using formatter.Formatxxx;
	// they will be written in front of msg
	fields    []func(f Formatter, event *Event) bytes.BufferWriter
	formatter Formatter
	writer    io.Writer

	// alarm and die are for panic and fatal
	// by default they will be panic(msg) and os.Exit(exitCode)
	// but in unit tests they can be replaced
	alarmPanic func(msg string)
	alarmFatal func(msg string, exitCode int)
}

func New() *Logger {
	return &Logger{
		level:      LevelInfo,
		callerSkip: 7,
		formatter:  &TextFormatter{},
		writer:     os.Stderr,
		alarmPanic: AlarmPanic,
		alarmFatal: AlarmFatal,
	}
}

func (l *Logger) SetLevel(level Level)   { l.level = level }
func (l *Logger) SetCallerSkip(skip int) { l.callerSkip = skip }
func (l *Logger) SetFields(fields ...func(f Formatter, event *Event) bytes.BufferWriter) {
	l.fields = fields
}
func (l *Logger) SetFormatter(formatter Formatter) { l.formatter = formatter }
func (l *Logger) SetWriter(writer io.Writer)       { l.writer = writer }
func (l *Logger) SetAlarmPanic(alarmPanic func(msg string)) {
	l.alarmPanic = alarmPanic
}
func (l *Logger) SetAlarmFatal(alarmFatal func(msg string, exitCode int)) {
	l.alarmFatal = alarmFatal
}

var (
	FieldLevel  = func(f Formatter, event *Event) bytes.BufferWriter { return f.FormatLevel(event.Level) }
	FieldTime   = func(f Formatter, event *Event) bytes.BufferWriter { return f.FormatTime(event.Time) }
	FieldCaller = func(f Formatter, event *Event) bytes.BufferWriter { return f.FormatCaller(event.CallerSkip) }
)

func (l *Logger) Debug(msg string, keyValues ...any) { l.log(LevelDebug, msg, keyValues...) }
func (l *Logger) Info(msg string, keyValues ...any)  { l.log(LevelInfo, msg, keyValues...) }
func (l *Logger) Warn(msg string, keyValues ...any)  { l.log(LevelWarn, msg, keyValues...) }
func (l *Logger) Error(msg string, keyValues ...any) { l.log(LevelError, msg, keyValues...) }
func (l *Logger) Panic(msg string, keyValues ...any) {
	l.log(LevelPanic, msg, keyValues...)
	l.alarmPanic(msg)
}

func (l *Logger) Fatal(msg string, keyValues ...any) {
	l.log(LevelFatal, msg, keyValues...)
	l.alarmFatal(msg, 1)
}

func (l *Logger) log(level Level, msg string, keyValues ...any) {
	if level < l.level {
		return
	}
	event := newEvent(level, l.callerSkip, time.Now())
	defer recycleEvent(event)
	var b bytes.Buffer
	b.ForNTimes(len(l.fields), func(b *bytes.Buffer, index int) {
		b.WriteFromBufferWriter(l.fields[index](l.formatter, event)).WriteString(l.formatter.MsgSplit())
	}).WriteFromBufferWriter(l.formatter.FormatMsg(msg, keyValues...)).WriteByte('\n')
	l.writer.Write(b.Bytes())
}

var (
	globalLock   sync.Mutex
	globalLogger *Logger = New()
)

// SetGlobalLogger sets a global logger.
//
// CallerSkip of global logger should be at least 7. Make sure that you set the correct caller skip.
func SetGlobalLogger(logger *Logger) {
	globalLock.Lock()
	globalLogger = logger
	globalLock.Unlock()
}

func Debug(msg string, keyValues ...any) { globalLogger.log(LevelDebug, msg, keyValues...) }
func Info(msg string, keyValues ...any)  { globalLogger.log(LevelInfo, msg, keyValues...) }
func Warn(msg string, keyValues ...any)  { globalLogger.log(LevelWarn, msg, keyValues...) }
func Error(msg string, keyValues ...any) { globalLogger.log(LevelError, msg, keyValues...) }

func Panic(msg string, keyValues ...any) {
	globalLogger.log(LevelPanic, msg, keyValues...)
	globalLogger.alarmPanic(msg)
}

func Fatal(msg string, keyValues ...any) {
	globalLogger.log(LevelFatal, msg, keyValues...)
	globalLogger.alarmFatal(msg, 1)
}
