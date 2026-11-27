// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package k8spod

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

// The operator image is distroless: it contains no shell and no coreutils (du, cat, ls, ...).
// ExecInDebugContainer runs commands in an ephemeral "debug" container instead, which has the same security context
// and mounts the same volumes under the same paths as the target container. Commands that only touch
// volume-backed paths (e.g. the CSI data directory) therefore see the same files as in the target container.
const (
	// DebugImageEnv overrides the image used for the debug container.
	DebugImageEnv = "E2E_DEBUG_IMAGE"

	// Index digest of busybox 1.37.0: covers amd64, arm64, ppc64le and s390x.
	// TODO: renovate?
	defaultDebugImage = "public.ecr.aws/docker/library/busybox:1.37.0@sha256:a7dbc319b1438115a207e11930dc4bb3e643dcb3ec0bdd694bb36cd8fad289ec"

	debugContainerPrefix       = "e2e-debug-"
	debugContainerSuffixLength = 5

	serviceAccountMountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
)

var (
	debugContainerStartTimeout = 5 * time.Minute // image pulls on the emulated clusters are slow
	debugContainerPollInterval = 2 * time.Second

	// The pod passed to ExecInDebugContainer is a stale copy, so every update starts with a fresh get and retries on conflicts.
	debugContainerUpdateBackoff = wait.Backoff{Steps: 10, Duration: 200 * time.Millisecond, Factor: 1.5, Jitter: 0.1}
)

// ExecInDebugContainer runs the command in a debug container of the pod, which is created on first use and reused afterwards.
func ExecInDebugContainer(ctx context.Context, resource *resources.Resources, pod *corev1.Pod, target string, command ...string) (*ExecutionResult, error) {
	// callers read the output even if the command failed, so never return a nil result
	emptyResult := &ExecutionResult{StdOut: &bytes.Buffer{}, StdErr: &bytes.Buffer{}}

	clientset, err := kubernetes.NewForConfig(resource.GetConfig())
	if err != nil {
		return emptyResult, errors.WithStack(err)
	}

	debugContainer, err := ensureDebugContainer(ctx, clientset, pod.Namespace, pod.Name, target)
	if err != nil {
		return emptyResult, err
	}

	return Exec(ctx, resource, pod, debugContainer, command...)
}

// ensureDebugContainer makes sure a running debug container exists in the pod for the target container and returns its name.
func ensureDebugContainer(ctx context.Context, clientset kubernetes.Interface, namespace, podName, target string) (string, error) {
	var name string

	err := retry.RetryOnConflict(debugContainerUpdateBackoff, func() error {
		current, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return err //nolint:wrapcheck // RetryOnConflict needs the original error
		}

		targetContainer := findContainer(current.Spec.Containers, target)
		if targetContainer == nil {
			return errors.Errorf("%q is not a regular container of pod %s/%s", target, namespace, podName)
		}

		if existing, found := findUsableDebugContainer(current, target); found {
			name = existing

			return nil
		}

		name = debugContainerPrefix + target + "-" + rand.String(debugContainerSuffixLength)

		updated := current.DeepCopy()
		updated.Spec.EphemeralContainers = append(updated.Spec.EphemeralContainers, newDebugContainer(name, *targetContainer))

		_, err = clientset.CoreV1().Pods(namespace).UpdateEphemeralContainers(ctx, podName, updated, metav1.UpdateOptions{})
		if err == nil {
			klog.InfoS("added debug container to run commands the target container cannot run", "pod", podName, "namespace", namespace, "target", target, "debugContainer", name)
		}

		return err //nolint:wrapcheck // RetryOnConflict needs the original error
	})
	if err != nil {
		return "", errors.WithMessagef(err, "failed to add debug container for %q to pod %s/%s", target, namespace, podName)
	}

	return name, waitForDebugContainer(ctx, clientset, namespace, podName, name)
}

func waitForDebugContainer(ctx context.Context, clientset kubernetes.Interface, namespace, podName, name string) error {
	lastState := "no status reported yet"

	err := wait.PollUntilContextTimeout(ctx, debugContainerPollInterval, debugContainerStartTimeout, true, func(ctx context.Context) (bool, error) {
		current, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			lastState = "failed to get the pod: " + err.Error()

			return false, nil //nolint:nilerr // transient, try again until the timeout
		}

		status := findEphemeralContainerStatus(current, name)

		switch {
		case status == nil:
			return false, nil
		case status.State.Running != nil:
			return true, nil
		case status.State.Terminated != nil:
			return false, errors.Errorf("terminated: %s %s", status.State.Terminated.Reason, status.State.Terminated.Message)
		case status.State.Waiting != nil:
			lastState = fmt.Sprintf("waiting: %s %s", status.State.Waiting.Reason, status.State.Waiting.Message)
		}

		return false, nil
	})
	if err != nil {
		return errors.WithMessagef(err, "debug container %q of pod %s/%s did not start (last state: %s)", name, namespace, podName, lastState)
	}

	return nil
}

// findUsableDebugContainer returns a debug container of the target container which has not terminated yet.
// A terminated ephemeral container is never restarted nor removed, so it has to be replaced by a new one.
func findUsableDebugContainer(pod *corev1.Pod, target string) (string, bool) {
	prefix := debugContainerPrefix + target + "-"

	for _, container := range pod.Spec.EphemeralContainers {
		suffix, found := strings.CutPrefix(container.Name, prefix)
		if !found || len(suffix) != debugContainerSuffixLength {
			continue
		}

		if status := findEphemeralContainerStatus(pod, container.Name); status != nil && status.State.Terminated != nil {
			continue
		}

		return container.Name, true
	}

	return "", false
}

func newDebugContainer(name string, target corev1.Container) corev1.EphemeralContainer {
	image := os.Getenv(DebugImageEnv)
	if image == "" {
		image = defaultDebugImage
	}

	return corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            name,
			Image:           image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			// "sleep" as PID 1 ignores SIGTERM, which would delay the deletion of the pod by the whole grace period.
			Command:         []string{"sh", "-c", "trap 'exit 0' TERM; sleep 86400 & wait"},
			SecurityContext: target.SecurityContext.DeepCopy(),
			VolumeMounts:    debugVolumeMounts(target.VolumeMounts),
		},
	}
}

// debugVolumeMounts copies the mounts of the target container as far as ephemeral containers are allowed to use them.
func debugVolumeMounts(targetMounts []corev1.VolumeMount) []corev1.VolumeMount {
	mounts := make([]corev1.VolumeMount, 0, len(targetMounts))

	for _, mount := range targetMounts {
		// not allowed in ephemeral containers
		if mount.SubPath != "" || mount.SubPathExpr != "" {
			continue
		}

		// added to every container (including ephemeral ones) by the admission, a copy would conflict
		if mount.MountPath == serviceAccountMountPath {
			continue
		}

		mounts = append(mounts, *mount.DeepCopy())
	}

	return mounts
}

func findContainer(containers []corev1.Container, name string) *corev1.Container {
	i := slices.IndexFunc(containers, func(c corev1.Container) bool { return c.Name == name })
	if i < 0 {
		return nil
	}

	return &containers[i]
}

func findEphemeralContainerStatus(pod *corev1.Pod, name string) *corev1.ContainerStatus {
	statuses := pod.Status.EphemeralContainerStatuses

	i := slices.IndexFunc(statuses, func(s corev1.ContainerStatus) bool { return s.Name == name })
	if i < 0 {
		return nil
	}

	return &statuses[i]
}
