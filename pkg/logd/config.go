package logd

import "sync"

const (
	stacktraceKey   = "stacktrace"
	errorVerboseKey = "errorVerbose"
)

type loggingConfig struct {
	LogLevel      LogLevel `json:"logLevel,omitempty"`
	LogEnterExits bool     `json:"LogEnterExits,omitempty"`
}

var (
	config     loggingConfig
	configOnce sync.Once
)

func init() {
	LoadConfig()
}

func LoadConfig() {
	configOnce.Do(func() {
		//TODO: read from ConfigMap
		config.LogEnterExits = true
		config.LogLevel = TraceLevel
	})
}
