package logger

import "log"

// LogFunc is alias of logger function.
type LogFunc = func(string, ...interface{})

// LogLevel is level of logger.
type LogLevel int8

const (
	// LogLevelDebug is DEBUG level.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is INFO level.
	LogLevelInfo
	// LogLevelWarn is WARN level.
	LogLevelWarn
	// LogLevelError is ERROR level.
	LogLevelError
)

var (
	lvl        = LogLevelInfo
	i, d, w, e = log.Printf, log.Printf, log.Printf, log.Printf
)

// SetLoggerLevel set global RSocket log level.
// Available levels are `LogLevelDebug`, `LogLevelInfo`, `LogLevelWarn` and `LogLevelError`.
func SetLoggerLevel(level LogLevel) {
	lvl = level
}

// SetLoggerDebug custom your debug log implementation.
func SetLoggerDebug(logFunc LogFunc) {
	d = logFunc
}

// SetLoggerInfo custom your info log implementation.
func SetLoggerInfo(logFunc LogFunc) {
	i = logFunc
}

// SetLoggerWarn custom your warn level log implementation.
func SetLoggerWarn(logFunc LogFunc) {
	w = logFunc
}

// SetLoggerError custom your error level log implementation.
func SetLoggerError(logFunc LogFunc) {
	e = logFunc
}

// IsDebugEnabled returns true if debug level is open.
func IsDebugEnabled() bool {
	return lvl <= LogLevelDebug
}

// Debugf prints debug level log.
func Debugf(format string, v ...interface{}) {
	if lvl > LogLevelDebug {
		return
	}
	d(format, v...)
}

// Infof prints info level log.
func Infof(format string, v ...interface{}) {
	if lvl > LogLevelInfo {
		return
	}
	i(format, v...)
}

// Warnf prints warn level log.
func Warnf(format string, v ...interface{}) {
	if lvl > LogLevelWarn {
		return
	}
	w(format, v...)
}

// Errorf prints error level log.
func Errorf(format string, v ...interface{}) {
	e(format, v...)
}
