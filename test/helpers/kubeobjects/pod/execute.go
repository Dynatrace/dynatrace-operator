//go:build e2e

package pod

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

type ExecutionResult struct {
	StdOut *bytes.Buffer
	StdErr *bytes.Buffer
}

func Exec(ctx context.Context, resource *resources.Resources, pod corev1.Pod, container string, command ...string) (*ExecutionResult, error) {
	result := &ExecutionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	err := resource.ExecInPod(
		ctx,
		pod.Namespace,
		pod.Name,
		container,
		command,
		result.StdOut,
		result.StdErr,
	)

	if err != nil {
		return result, errors.WithMessagef(errors.WithStack(err),
			"stdout:\n%s\nstderr:\n%s", result.StdOut.String(), result.StdErr.String())
	}

	return result, nil
}
