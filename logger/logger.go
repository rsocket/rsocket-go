package logger

import (
	"log"
	"sync"
)

var (
	_mu     sync.RWMutex
	_level         = LevelInfo
	_logger Logger = simpleLogger{}
)

const (
	// LevelDebug is DEBUG level.
	LevelDebug Level = 1 << iota
	// LevelInfo is INFO level.
	LevelInfo
	// LevelWarn is WARN level.
	LevelWarn
	// LevelError is ERROR level.
	LevelError
)

// Logger is used to print logs.
type Logger interface {
	// Debugf print to the debug level logs.
	Debugf(format string, args ...interface{})
	// Infof print to the info level logs.
	Infof(format string, args ...interface{})
	// Warnf print to the info level logs.
	Warnf(format string, args ...interface{})
	// Errorf print to the info level logs.
	Errorf(format string, args ...interface{})
}

// Level is level of logger.
type Level int8

// SetLevel set global RSocket log level.
// Available levels are `LevelDebug`, `LevelInfo`, `LevelWarn` and `LevelError`.
func SetLevel(level Level) {
	_mu.Lock()
	defer _mu.Unlock()
	_level = level
}

// SetLogger customize the global logger.
// A standard log implementation will be used by default.
func SetLogger(logger Logger) {
	_mu.Lock()
	defer _mu.Unlock()
	_logger = logger
}

// GetLevel returns current logger level.
func GetLevel() Level {
	_mu.RLock()
	defer _mu.RUnlock()
	return _level
}

// IsDebugEnabled returns true if debug level is open.
func IsDebugEnabled() bool {
	_mu.RLock()
	defer _mu.RUnlock()
	return _level <= LevelDebug
}

// Debugf prints debug level log.
func Debugf(format string, args ...interface{}) {
	logger, level := getLoggerAndLevel()
	if logger == nil || level > LevelDebug {
		return
	}
	logger.Debugf(format, args...)
}

// Infof prints info level log.
func Infof(format string, args ...interface{}) {
	logger, level := getLoggerAndLevel()
	if logger == nil || level > LevelInfo {
		return
	}
	logger.Infof(format, args...)
}

// Warnf prints warn level log.
func Warnf(format string, args ...interface{}) {
	logger, level := getLoggerAndLevel()
	if logger == nil || level > LevelWarn {
		return
	}
	logger.Warnf(format, args...)
}

// Errorf prints error level log.
func Errorf(format string, args ...interface{}) {
	logger, level := getLoggerAndLevel()
	if logger == nil || level > LevelError {
		return
	}
	logger.Errorf(format, args...)
}

func getLoggerAndLevel() (Logger, Level) {
	_mu.RLock()
	defer _mu.RUnlock()
	return _logger, _level
}

type simpleLogger struct {
}

func (s simpleLogger) Debugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

func (s simpleLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (s simpleLogger) Warnf(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

func (s simpleLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}
