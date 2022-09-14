package pod

import (
	"bytes"
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
	command   string
	container string
}

func NewExecutionQuery(pod v1.Pod, container string, command string) ExecutionQuery {
	return ExecutionQuery{
		pod:       pod,
		command:   command,
		container: container,
	}
}

func (query ExecutionQuery) Execute(client kubernetes.Interface, restConfig *rest.Config) (*ExecutionResult, error) {
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
		Tty:    true,
	})

	return result, errors.WithStack(err)
}

func (query ExecutionQuery) buildExecutionOptions() *v1.PodExecOptions {
	return &v1.PodExecOptions{
		Command:   query.buildCommands(),
		Container: query.container,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}
}

func (query ExecutionQuery) buildRequest(client kubernetes.Interface) *rest.Request {
	return client.CoreV1().RESTClient().Post().Resource(resourcePods).
		Namespace(query.pod.Namespace).Name(query.pod.Name).SubResource(resourceExec).
		VersionedParams(query.buildExecutionOptions(), scheme.ParameterCodec)
}

func (query ExecutionQuery) buildCommands() []string {
	return []string{
		"sh", "-c", query.command,
	}
}
