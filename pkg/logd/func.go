package logd

const dtErrorCodeFieldName = "dt-error-code"

func (l Logger) Error(err error, msg string, keysAndValues ...any) {
	Error(l.Logger, err, msg, keysAndValues...)
}

func (l Logger) ErrorCode(err error, msg string, errCode string, keysAndValues ...any) {
	Error(l.Logger.WithValues(dtErrorCodeFieldName, errCode), err, msg, keysAndValues...)
}

func (l Logger) Warn(msg string, keysAndValues ...any) {
	Warn(l.Logger, msg, keysAndValues...)
}

func (l Logger) Info(msg string, keysAndValues ...any) {
	Info(l.Logger, msg, keysAndValues...)
}

func (l Logger) Debug(msg string, keysAndValues ...any) {
	Debug(l.Logger, msg, keysAndValues...)
}

func (l Logger) Trace(msg string, keysAndValues ...any) {
	Trace(l.Logger, msg, keysAndValues...)
}

func (l Logger) Enter(scope string, keysAndValues ...any) {
	if config.LogEnterExits {
		l.WithValues("scope", scope).Info("Enter "+scope, keysAndValues...)
	}
}

func (l Logger) ExitSuccess(scope string, result string, keysAndValues ...any) {
	if config.LogEnterExits {
		l.WithValues("scope", scope, "result", "success: "+result).Info("Exit "+scope+" successfully", keysAndValues...)
	}
}

func (l Logger) ExitFail(message string, keysAndValues ...any) {
	l.ExitFailNg(message, "", keysAndValues...)
}

func (l Logger) ExitFailNg(scope string, result string, keysAndValues ...any) {
	if config.LogEnterExits {
		l.WithValues("scope", scope, "result", "failed: "+result).Info("Exit "+scope+" with failure", keysAndValues...)
	}
}

func (l Logger) ExitError(err error, message string, keysAndValues ...any) {
	l.ExitErrorNg(err, message, "", keysAndValues...)
}

func (l Logger) ExitErrorNg(err error, scope string, result string, keysAndValues ...any) {
	if config.LogEnterExits {
		l.WithValues("scope", scope, "result", "failed: "+result).Error(err, "Exit "+scope+" with error", keysAndValues...)
	}
}
