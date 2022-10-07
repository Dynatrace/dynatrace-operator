package logger

import "github.com/go-logr/logr"

var Factory factory

func init() {
	Factory = factory{logger: newLogger()}
}

type factory struct {
	logger logr.Logger
}

func (f factory) GetLogger(name string) logr.Logger {
	return f.logger.WithName(name)
}
