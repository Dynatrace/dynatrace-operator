package remote_command

import (
	"bytes"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Executor interface {
	Exec(ctx context.Context, config *rest.Config, podName string, podNamespace string, containerName string, command []string) (stdOut *bytes.Buffer, stdErr *bytes.Buffer, err error)
}

type DefaultExecutor struct{}

func (r DefaultExecutor) Exec(ctx context.Context, config *rest.Config, podName string, podNamespace string, containerName string, command []string) (stdOut *bytes.Buffer, stdErr *bytes.Buffer, err error) { //nolint:revive
	sch := scheme.Scheme
	parameterCodec := scheme.ParameterCodec

	gvk := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}

	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	restClient, err := apiutil.RESTClientForGVK(gvk, false, config, serializer.NewCodecFactory(sch), httpClient)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	stdOut = &bytes.Buffer{}
	stdErr = &bytes.Buffer{}

	req := restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(podNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, parameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  &bytes.Buffer{},
		Stdout: stdOut,
		Stderr: stdErr,
		Tty:    false,
	})

	return stdOut, stdErr, errors.WithStack(err)
}
