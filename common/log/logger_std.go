package log

import (
	"fmt"
	"log"
)

type LoggerStd struct {
}

func (logger *LoggerStd) Log(lv LogLevel, args ...interface{}) {
	_ = log.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprint(args...)+"\n")
}

func (logger *LoggerStd) Logf(lv LogLevel, format string, args ...interface{}) {
	_ = log.Output(2, fmt.Sprintf("[%s]", lv)+fmt.Sprintf(format, args...)+"\n")
}
