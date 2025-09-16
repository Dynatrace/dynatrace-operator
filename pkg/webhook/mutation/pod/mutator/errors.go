package mutator

import corev1 "k8s.io/api/core/v1"

//nolint:revive // The mutator package is always aliased so this name doesn't stutter.
type MutatorError struct {
	Err      error
	Annotate func(*corev1.Pod)
}

func (e MutatorError) Error() string {
	return e.Err.Error()
}

func (e MutatorError) Unwrap() error {
	return e.Err
}

func (e MutatorError) SetAnnotations(pod *corev1.Pod) {
	if e.Annotate != nil {
		e.Annotate(pod)
	}
}
