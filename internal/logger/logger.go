package logger

import (
	"fmt"
	"log"
)

// Func is alias of logger function.
type Func = func(string, ...interface{})

// Level is level of logger.
type Level int8

func (s Level) String() string {
	switch s {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

const (
	// LevelDebug is DEBUG level.
	LevelDebug Level = iota
	// LevelInfo is INFO level.
	LevelInfo
	// LevelWarn is WARN level.
	LevelWarn
	// LevelError is ERROR level.
	LevelError
)

var (
	lvl        = LevelInfo
	i, d, w, e = log.Printf, log.Printf, log.Printf, log.Printf
	prefix     = true
)

// SetLevel set global RSocket log level.
// Available levels are `LogLevelDebug`, `LogLevelInfo`, `LogLevelWarn` and `LogLevelError`.
func SetLevel(level Level) {
	lvl = level
}

// DisablePrefix disable print level prefix.
func DisablePrefix() {
	prefix = false
}

// GetLevel returns current logger level.
func GetLevel() Level {
	return lvl
}

// SetFunc set logger func for custom level.
func SetFunc(level Level, fn Func) {
	if fn == nil {
		return
	}
	switch level {
	case LevelDebug:
		d = fn
	case LevelInfo:
		i = fn
	case LevelWarn:
		w = fn
	case LevelError:
		e = fn
	}
}

// IsDebugEnabled returns true if debug level is open.
func IsDebugEnabled() bool {
	return lvl <= LevelDebug
}

// Debugf prints debug level log.
func Debugf(format string, v ...interface{}) {
	if lvl > LevelDebug {
		return
	}
	if prefix {
		d(fmt.Sprintf("[%s] %s", LevelDebug, format), v...)
	} else {
		d(format, v...)
	}
}

// Infof prints info level log.
func Infof(format string, v ...interface{}) {
	if lvl > LevelInfo {
		return
	}
	if prefix {
		i(fmt.Sprintf("[%s] %s", LevelInfo, format), v...)
	} else {
		i(format, v...)
	}
}

// Warnf prints warn level log.
func Warnf(format string, v ...interface{}) {
	if lvl > LevelWarn {
		return
	}
	if prefix {
		w(fmt.Sprintf("[%s] %s", LevelWarn, format), v...)
	} else {
		w(format, v...)
	}
}

// Errorf prints error level log.
func Errorf(format string, v ...interface{}) {
	if prefix {
		e(fmt.Sprintf("[%s] %s", LevelError, format), v...)
	} else {
		e(format, v...)
	}
}
