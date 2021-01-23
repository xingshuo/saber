package log

type LogSystem struct {
	logger Logger
	level  LogLevel
}

// SetLogger 设置默认logger
func (s *LogSystem) SetLogger(logger Logger) {
	s.logger = logger
}

func (s *LogSystem) GetLogger() Logger {
	return s.logger
}

func (s *LogSystem) SetLevel(level LogLevel) {
	s.level = level
}

func (s *LogSystem) GetLevel() LogLevel {
	return s.level
}

func (s *LogSystem) Debug(args ...interface{}) {
	if s.level > LevelDebug {
		return
	}
	s.logger.Log(LevelDebug, args...)
}

func (s *LogSystem) Debugf(format string, args ...interface{}) {
	if s.level > LevelDebug {
		return
	}
	s.logger.Logf(LevelDebug, format, args...)
}

func (s *LogSystem) Info(args ...interface{}) {
	if s.level > LevelInfo {
		return
	}
	s.logger.Log(LevelInfo, args...)
}

func (s *LogSystem) Infof(format string, args ...interface{}) {
	if s.level > LevelInfo {
		return
	}
	s.logger.Logf(LevelInfo, format, args...)
}

func (s *LogSystem) Warning(args ...interface{}) {
	if s.level > LevelWarning {
		return
	}
	s.logger.Log(LevelWarning, args...)
}

func (s *LogSystem) Warningf(format string, args ...interface{}) {
	if s.level > LevelWarning {
		return
	}
	s.logger.Logf(LevelWarning, format, args...)
}

func (s *LogSystem) Error(args ...interface{}) {
	if s.level > LevelError {
		return
	}
	s.logger.Log(LevelError, args...)
}

func (s *LogSystem) Errorf(format string, args ...interface{}) {
	if s.level > LevelError {
		return
	}
	s.logger.Logf(LevelError, format, args...)
}

func NewLogSystem(logger Logger, level LogLevel) *LogSystem {
	return &LogSystem{
		logger: logger,
		level:  level,
	}
}

func NewStdLogSystem(level LogLevel) *LogSystem {
	return &LogSystem{
		logger: &LoggerStd{},
		level:  level,
	}
}
