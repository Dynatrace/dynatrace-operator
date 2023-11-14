package errors

var _ error = (*RestartReconciliationError)(nil)

type RestartReconciliationError struct {
	message string
}

func NewRestartReconciliationError(message string) error {
	return &RestartReconciliationError{message: message}
}

func (e *RestartReconciliationError) Error() string {
	return e.message
}

func (e *RestartReconciliationError) Is(target error) bool {
	_, ok := target.(*RestartReconciliationError)
	return ok
}
