package log

import (
	"fmt"
	"log"
)

type LoggerStd struct {
	level LogLevel
}

func (logger *LoggerStd) Log(lv LogLevel, args ...interface{}) {
	if logger.level > lv {
		return
	}
	_ = log.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprint(args...)+"\n")
}

func (logger *LoggerStd) Logf(lv LogLevel, format string, args ...interface{}) {
	if logger.level > lv {
		return
	}
	_ = log.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprintf(format, args...)+"\n")
}

func (logger *LoggerStd) SetLevel(level LogLevel) {
	logger.level = level
}

func (logger *LoggerStd) GetLevel() LogLevel {
	return logger.level
}
