package log

import "sync"

var (
	DefaultLogger Logger
	initLog       sync.Once
)

//日志默认为stdout
func init() {
	initLog.Do(func() {
		DefaultLogger = &LoggerStd{
			level: LevelInfo,
		}
	})
}

// SetLogger 设置默认logger
func SetLogger(logger Logger) {
	DefaultLogger = logger
}

func Debug(args ...interface{}) {
	DefaultLogger.Log(LevelDebug, args...)
}

func Debugf(format string, args ...interface{}) {
	DefaultLogger.Logf(LevelDebug, format, args...)
}

func Info(args ...interface{}) {
	DefaultLogger.Log(LevelInfo, args...)
}

func Infof(format string, args ...interface{}) {
	DefaultLogger.Logf(LevelInfo, format, args...)
}

func Warning(args ...interface{}) {
	DefaultLogger.Log(LevelWarning, args...)
}

func Warningf(format string, args ...interface{}) {
	DefaultLogger.Logf(LevelWarning, format, args...)
}

func Error(args ...interface{}) {
	DefaultLogger.Log(LevelError, args...)
}

func Errorf(format string, args ...interface{}) {
	DefaultLogger.Logf(LevelError, format, args...)
}
