package log

type LogLevel int64

const (
	LevelDebug = iota
	LevelInfo
	LevelWarning
	LevelError
)

func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOW"
	}
}

type Logger interface {
	Log(lv LogLevel, args ...interface{})
	Logf(lv LogLevel, format string, args ...interface{})
}
