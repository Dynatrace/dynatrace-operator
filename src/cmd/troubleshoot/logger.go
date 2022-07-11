package troubleshoot

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	Prefix string
	Logger *log.Logger
}

func NewLogger(prefix string) *Logger {
	return &Logger{
		prefix,
		log.New(os.Stdout, prefix, 0),
	}
}

func (logger *Logger) SetPrefix(prefix string) {
	logger.Logger.SetPrefix(prefix)
}

func (logger *Logger) NewTestf(format string, v ...interface{}) {
	logger.Logger.Printf("--- "+format, v...)
}

func (logger *Logger) Infof(format string, v ...interface{}) {
	logger.Logger.Printf("    "+format, v...)
}

func (logger *Logger) Okf(format string, v ...interface{}) {
	logger.Logger.Printf(" \u221A  "+format, v...)
}

func (logger *Logger) Errorf(format string, v ...interface{}) {
	logger.Logger.Printf(" \u00D7  "+format, v...)
}

func (logger *Logger) WithErrorf(err error, format string, v ...interface{}) {
	message := fmt.Sprintf(" \u00D7  "+format, v...)
	logger.Logger.Printf("%s (%s)", message, err.Error())
}
