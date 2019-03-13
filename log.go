package rsocket

import (
	"log"
)

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

var logger loggerR = &loggerWrapper{
	lvl: LogLevelInfo,
	d:   log.Printf,
	i:   log.Printf,
	w:   log.Printf,
	e:   log.Printf,
}

// SetLoggerLevel set global RSocket log level.
// Available levels are `LogLevelDebug`, `LogLevelInfo`, `LogLevelWarn` and `LogLevelError`.
func SetLoggerLevel(level LogLevel) {
	logger.(*loggerWrapper).lvl = level
}

// SetLoggerDebug custom your debug log implementation.
func SetLoggerDebug(logFunc LogFunc) {
	logger.(*loggerWrapper).d = logFunc
}

// SetLoggerInfo custom your info log implementation.
func SetLoggerInfo(logFunc LogFunc) {
	logger.(*loggerWrapper).i = logFunc
}

// SetLoggerWarn custom your warn level log implementation.
func SetLoggerWarn(logFunc LogFunc) {
	logger.(*loggerWrapper).w = logFunc
}

// SetLoggerError custom your error level log implementation.
func SetLoggerError(logFunc LogFunc) {
	logger.(*loggerWrapper).e = logFunc
}

type loggerR interface {
	IsDebugEnabled() bool
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

type loggerWrapper struct {
	lvl        LogLevel
	d, i, w, e LogFunc
}

func (p *loggerWrapper) IsDebugEnabled() bool {
	return p.lvl <= LogLevelDebug
}

func (p *loggerWrapper) Debugf(format string, v ...interface{}) {
	if p.lvl > LogLevelDebug {
		return
	}
	p.d(format, v...)
}

func (p *loggerWrapper) Infof(format string, v ...interface{}) {
	if p.lvl > LogLevelInfo {
		return
	}
	p.i(format, v...)
}

func (p *loggerWrapper) Warnf(format string, v ...interface{}) {
	if p.lvl > LogLevelWarn {
		return
	}
	p.w(format, v...)
}

func (p *loggerWrapper) Errorf(format string, v ...interface{}) {
	p.e(format, v...)
}
