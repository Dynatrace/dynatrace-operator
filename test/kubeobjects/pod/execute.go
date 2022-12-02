package pod

import (
	"bytes"
	"github.com/Dynatrace/dynatrace-operator/test/shell"
	"net/http"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	resourcePods = "pods"
	resourceExec = "exec"
)

type ExecutionResult struct {
	StdOut *bytes.Buffer
	StdErr *bytes.Buffer
}

type ExecutionQuery struct {
	pod       v1.Pod
	command   shell.Command
	container string
	tty       bool
}

func NewExecutionQuery(pod v1.Pod, container string, command ...string) ExecutionQuery {
	query := ExecutionQuery{
		pod:       pod,
		container: container,
		command:   make([]string, 0),
		tty:       false,
	}
	query.command = append(query.command, command...)
	return query
}

func (query ExecutionQuery) WithTTY(tty bool) ExecutionQuery {
	query.tty = tty
	return query
}

func (query ExecutionQuery) Execute(restConfig *rest.Config) (*ExecutionResult, error) {
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req := query.buildRequest(client)
	executor, err := remotecommand.NewSPDYExecutor(restConfig, http.MethodPost, req.URL())

	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := &ExecutionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: result.StdOut,
		Stderr: result.StdErr,
		Tty:    query.tty,
	})

	if err != nil {
		return result, errors.WithMessagef(errors.WithStack(err),
			"stdout:\n%s\nstderr:\n%s", result.StdOut.String(), result.StdErr.String())
	}

	return result, nil
}

func (query ExecutionQuery) buildExecutionOptions() *v1.PodExecOptions {
	return &v1.PodExecOptions{
		Command:   query.command,
		Container: query.container,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       query.tty,
	}
}

func (query ExecutionQuery) buildRequest(client kubernetes.Interface) *rest.Request {
	return client.CoreV1().RESTClient().Post().Resource(resourcePods).
		Namespace(query.pod.Namespace).Name(query.pod.Name).SubResource(resourceExec).
		VersionedParams(query.buildExecutionOptions(), scheme.ParameterCodec)
}
