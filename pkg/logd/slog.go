package logd

/*
func NewSlogger() logr.Logger {
	// make dynamic changing of log level possible
	var lv slog.LevelVar
	lv.Set(slog.LevelInfo)

	handlerOpts := &slog.HandlerOptions{
		AddSource: false,
		Level:     &lv,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// adjust keys to be Dynatrace Logging Technology compliant
			switch a.Key {
			case slog.TimeKey:
				// adjust timestamp format
				t := a.Value.Time()

				return slog.String("timestamp", t.Format(time.RFC3339))
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				// this one is already DT compliant
				a.Key = slog.LevelKey
			}

			return a
		},
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, handlerOpts)

	return logr.FromSlogHandler(jsonHandler)
}
*/
