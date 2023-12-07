package log

import "errors"

type Level uint8

var ErrLevelAlreadyRegistered = errors.New("level already registered")

const (
	LevelDebug Level = 10 * (1 + iota)
	LevelInfo
	LevelWarn
	LevelError
	LevelPanic
	LevelFatal

	LevelUnknown Level = 200
)

func (l Level) String() string {
	if s, ok := levelMap[l]; ok {
		return s
	}
	return levelMap[LevelUnknown]
}

func RegisterLevel(l Level, s string) error {
	if _, exist := levelMap[l]; exist {
		return ErrLevelAlreadyRegistered
	}
	levelMap[l] = s
	return nil
}

var (
	levelMap = map[Level]string{
		LevelDebug:   "debug",
		LevelInfo:    "info",
		LevelWarn:    "warn",
		LevelError:   "error",
		LevelPanic:   "panic",
		LevelFatal:   "fatal",
		LevelUnknown: "unknown",
	}
)
