package logd

const (
	stacktraceKey   = "stacktrace"
	errorVerboseKey = "errorVerbose"
)

type loggingConfig struct {
	LogLevel      LogLevel `json:"logLevel,omitempty"`
	LogEnterExits bool     `json:"LogEnterExits,omitempty"`
}

var (
	config loggingConfig
	// configOnce sync.Once
)

/*
func LoadConfig() {
	configOnce.Do(func() {
		// read from ConfigMap
	})
}
*/
