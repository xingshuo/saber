package log

type LoggerWrapper struct {
	logger Logger
}

// SetLogger 设置默认logger
func (lw *LoggerWrapper) SetLogger(logger Logger) {
	lw.logger = logger
}

func (lw *LoggerWrapper) GetLogger() Logger {
	return lw.logger
}

func (lw *LoggerWrapper) Debug(args ...interface{}) {
	lw.logger.Log(LevelDebug, args...)
}

func (lw *LoggerWrapper) Debugf(format string, args ...interface{}) {
	lw.logger.Logf(LevelDebug, format, args...)
}

func (lw *LoggerWrapper) Info(args ...interface{}) {
	lw.logger.Log(LevelInfo, args...)
}

func (lw *LoggerWrapper) Infof(format string, args ...interface{}) {
	lw.logger.Logf(LevelInfo, format, args...)
}

func (lw *LoggerWrapper) Warning(args ...interface{}) {
	lw.logger.Log(LevelWarning, args...)
}

func (lw *LoggerWrapper) Warningf(format string, args ...interface{}) {
	lw.logger.Logf(LevelWarning, format, args...)
}

func (lw *LoggerWrapper) Error(args ...interface{}) {
	lw.logger.Log(LevelError, args...)
}

func (lw *LoggerWrapper) Errorf(format string, args ...interface{}) {
	lw.logger.Logf(LevelError, format, args...)
}

func NewLoggerWrapper() *LoggerWrapper {
	return &LoggerWrapper{
		logger: &LoggerStd{
			level: LevelInfo,
		},
	}
}
