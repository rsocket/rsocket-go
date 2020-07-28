package logger

import "log"

var (
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
	_level = level
}

// SetLogger customize the global logger.
// A standard log implementation will be used by default.
func SetLogger(logger Logger) {
	_logger = logger
}

// GetLevel returns current logger level.
func GetLevel() Level {
	return _level
}

// IsDebugEnabled returns true if debug level is open.
func IsDebugEnabled() bool {
	return _level <= LevelDebug
}

// Debugf prints debug level log.
func Debugf(format string, args ...interface{}) {
	if _logger == nil || _level > LevelDebug {
		return
	}
	_logger.Debugf(format, args...)
}

// Infof prints info level log.
func Infof(format string, args ...interface{}) {
	if _logger == nil || _level > LevelInfo {
		return
	}
	_logger.Infof(format, args...)
}

// Warnf prints warn level log.
func Warnf(format string, args ...interface{}) {
	if _logger == nil || _level > LevelWarn {
		return
	}
	_logger.Warnf(format, args...)
}

// Errorf prints error level log.
func Errorf(format string, args ...interface{}) {
	if _logger == nil || _level > LevelError {
		return
	}
	_logger.Errorf(format, args...)
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
