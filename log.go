package rsocket

import (
	"log"
)

type LogFunc = func(string, ...interface{})

type LogLevel int8

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var logger Logger = &loggerWrapper{
	lvl: LogLevelInfo,
	d:   log.Printf,
	i:   log.Printf,
	w:   log.Printf,
	e:   log.Printf,
}

func SetLoggerLevel(level LogLevel) {
	logger.(*loggerWrapper).lvl = level
}

func SetLoggerDebug(logFunc LogFunc) {
	logger.(*loggerWrapper).d = logFunc
}

func SetLoggerInfo(logFunc LogFunc) {
	logger.(*loggerWrapper).i = logFunc
}

func SetLoggerWarn(logFunc LogFunc) {
	logger.(*loggerWrapper).w = logFunc
}

func SetLoggerError(logFunc LogFunc) {
	logger.(*loggerWrapper).e = logFunc
}

type Logger interface {
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
