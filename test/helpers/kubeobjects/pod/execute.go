//go:build e2e

package pod

import (
	"bytes"
	"context"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

type ExecutionResult struct {
	StdOut *bytes.Buffer
	StdErr *bytes.Buffer
}

type ExecutionQuery struct {
	ctx       context.Context
	resource  *resources.Resources
	pod       corev1.Pod
	command   shell.Command
	container string
}

func NewExecutionQuery(ctx context.Context, resource *resources.Resources, pod corev1.Pod, container string, command ...string) ExecutionQuery {
	query := ExecutionQuery{
		ctx:       ctx,
		resource:  resource,
		pod:       pod,
		container: container,
		command:   make([]string, 0),
	}
	query.command = append(query.command, command...)
	return query
}

func (query ExecutionQuery) Execute() (*ExecutionResult, error) {
	result := &ExecutionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	err := query.resource.ExecInPod(
		query.ctx, query.pod.Namespace, query.pod.Name, query.container, query.command, result.StdOut, result.StdErr)

	if err != nil {
		return result, errors.WithMessagef(errors.WithStack(err),
			"stdout:\n%s\nstderr:\n%s", result.StdOut.String(), result.StdErr.String())
	}

	return result, nil
}
