// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package nodes

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// imageVolumeMinK8s is the minimum Kubernetes version that supports image volumes.
var imageVolumeMinK8s = version.MustParseSemantic("v1.33.0")

// imageVolumeMinRuntimes maps runtime type to minimum required version.
var imageVolumeMinRuntimes = map[string]*version.Version{
	"containerd": version.MustParseSemantic("v2.2.0"),
	"cri-o":      version.MustParseSemantic("v1.33.0"),
}

// ContainerRuntime holds parsed container runtime info.
type ContainerRuntime struct {
	// Type is "containerd", "cri-o", etc.
	Type    string
	Version *version.Version
}

// parseKubeletVersion strips distro suffixes (e.g. "-gke.2064000") and parses
// the semantic version.
// Input:
// v1.30.14
func parseKubeletVersion(kubeletVersion string) (*version.Version, error) {
	parsed, err := version.ParseSemantic(kubeletVersion)
	if err != nil {
		return nil, fmt.Errorf("parse kubelet version %q: %w", kubeletVersion, err)
	}

	return parsed, nil
}

// parseContainerRuntime splits "containerd://2.2.2" or "cri-o://1.28.0" into
// runtime type and version.
func parseContainerRuntime(runtimeVersion string) (*ContainerRuntime, error) {
	before, after, found := strings.Cut(runtimeVersion, "://")
	if !found {
		return nil, fmt.Errorf("unexpected container runtime format: %q", runtimeVersion)
	}

	parsed, err := version.ParseSemantic(after)
	if err != nil {
		return nil, fmt.Errorf("parse runtime version %q: %w", after, err)
	}

	return &ContainerRuntime{Type: before, Version: parsed}, nil
}

// nodeSupportsImageVolumes returns true when both the kubelet and the container
// runtime on node meet the minimum versions required for image volumes.
func nodeSupportsImageVolumes(k8sVer *version.Version, rt *ContainerRuntime) (bool, error) {
	if !k8sVer.AtLeast(imageVolumeMinK8s) {
		return false, nil
	}

	minRT, known := imageVolumeMinRuntimes[rt.Type]
	if !known {
		return false, fmt.Errorf("unknown container runtime type: %q", rt.Type)
	}

	return rt.Version.AtLeast(minRT), nil
}

// IsImageVolumesSupported returns true when every node in the cluster
// meets the minimum k8s and container-runtime versions for image volumes.
func IsImageVolumesSupported(t *testing.T) bool {
	t.Helper()

	restCfg, err := config.GetConfig()
	require.NoError(t, err, "get kubeconfig")

	clientset, err := kubernetes.NewForConfig(restCfg)
	require.NoError(t, err, "create k8s clientset")

	nodeList, err := clientset.CoreV1().Nodes().List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err, "list nodes")

	for _, node := range nodeList.Items {
		k8sVer, err := parseKubeletVersion(node.Status.NodeInfo.KubeletVersion)
		if err != nil {
			t.Fatalf("parse kubelet version %q: %v", node.Status.NodeInfo.KubeletVersion, err)

			return false
		}

		rt, err := parseContainerRuntime(node.Status.NodeInfo.ContainerRuntimeVersion)
		if err != nil {
			t.Fatalf("parse container runtime %q: %v", node.Status.NodeInfo.ContainerRuntimeVersion, err)

			return false
		}

		t.Logf("detected node %-40s  k8s=%s  runtime=%s runtime version=%s",
			node.Name, k8sVer, rt.Type, rt.Version)

		ok, err := nodeSupportsImageVolumes(k8sVer, rt)
		if err != nil {
			t.Logf("node %s: %v", node.Name, err)

			return false
		}

		if !ok {
			return false
		}
	}

	return true
}
