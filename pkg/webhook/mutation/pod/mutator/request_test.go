package mutator

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodName(t *testing.T) {
	t.Run("returns empty string when pod is nil", func(t *testing.T) {
		req := &BaseRequest{
			Pod: nil,
		}
		assert.Equal(t, "", req.PodName())
	})

	t.Run("returns pod name when set", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
		}
		assert.Equal(t, "test-pod", req.PodName())
	})

	t.Run("returns generate name when name is empty", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-pod-",
				},
			},
		}
		assert.Equal(t, "test-pod-", req.PodName())
	})

	t.Run("prefers name over generate name", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "test-pod",
					GenerateName: "test-pod-",
				},
			},
		}
		assert.Equal(t, "test-pod", req.PodName())
	})
}

func TestIsSplitMountsEnabled(t *testing.T) {
	t.Run("returns true when annotation is set to 'true'", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationInjectionSplitMounts: "true",
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}
		assert.True(t, req.IsSplitMountsEnabled())
	})

	t.Run("returns true when annotation is set to 'True' (case insensitive)", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationInjectionSplitMounts: "True",
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}
		assert.True(t, req.IsSplitMountsEnabled())
	})

	t.Run("returns true when annotation is set to 'TRUE' (case insensitive)", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationInjectionSplitMounts: "TRUE",
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}
		assert.True(t, req.IsSplitMountsEnabled())
	})

	t.Run("returns false when annotation is set to 'false'", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationInjectionSplitMounts: "false",
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}
		assert.False(t, req.IsSplitMountsEnabled())
	})

	t.Run("returns false when annotation is missing and DynaKube is not classic mode", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
					},
				},
			},
		}
		assert.False(t, req.IsSplitMountsEnabled())
	})

	t.Run("returns true when annotation is missing but DynaKube is in classic mode", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					OneAgent: oneagent.Spec{
						ClassicFullStack: &oneagent.HostInjectSpec{},
					},
				},
			},
		}
		assert.True(t, req.IsSplitMountsEnabled())
	})

	t.Run("annotation 'true' takes precedence over DynaKube mode", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationInjectionSplitMounts: "true",
					},
				},
			},
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					OneAgent: oneagent.Spec{
						CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
					},
				},
			},
		}
		assert.True(t, req.IsSplitMountsEnabled())
	})

	t.Run("annotation 'false' falls back to DynaKube mode", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationInjectionSplitMounts: "false",
					},
				},
			},
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					OneAgent: oneagent.Spec{
						ClassicFullStack: &oneagent.HostInjectSpec{},
					},
				},
			},
		}
		assert.True(t, req.IsSplitMountsEnabled())
	})
}

func TestNewContainers(t *testing.T) {
	alwaysFalse := func(corev1.Container, *BaseRequest) bool {
		return false
	}

	alwaysTrue := func(corev1.Container, *BaseRequest) bool {
		return true
	}

	t.Run("returns all containers when none excluded and none injected", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container1"},
						{Name: "container2"},
						{Name: "container3"},
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}

		newContainers := req.NewContainers(alwaysFalse)
		require.Len(t, newContainers, 3)
		assert.Equal(t, "container1", newContainers[0].Name)
		assert.Equal(t, "container2", newContainers[1].Name)
		assert.Equal(t, "container3", newContainers[2].Name)
	})

	t.Run("returns empty list when all containers are already injected", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container1"},
						{Name: "container2"},
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}

		newContainers := req.NewContainers(alwaysTrue)
		assert.Empty(t, newContainers)
	})

	t.Run("excludes containers based on pod annotations", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationContainerInjection + "/container2": "false",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container1"},
						{Name: "container2"},
						{Name: "container3"},
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}

		newContainers := req.NewContainers(alwaysFalse)
		require.Len(t, newContainers, 2)
		assert.Equal(t, "container1", newContainers[0].Name)
		assert.Equal(t, "container3", newContainers[1].Name)
	})

	t.Run("excludes containers based on dynakube annotations", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container1"},
						{Name: "container2"},
						{Name: "container3"},
					},
				},
			},
			DynaKube: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationContainerInjection + "/container1": "false",
					},
				},
			},
		}

		newContainers := req.NewContainers(alwaysFalse)
		require.Len(t, newContainers, 2)
		assert.Equal(t, "container2", newContainers[0].Name)
		assert.Equal(t, "container3", newContainers[1].Name)
	})

	t.Run("excludes containers based on isInjected predicate", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container1"},
						{Name: "container2"},
						{Name: "container3"},
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}

		isInjected := func(c corev1.Container, _ *BaseRequest) bool {
			return c.Name == "container2"
		}

		newContainers := req.NewContainers(isInjected)
		require.Len(t, newContainers, 2)
		assert.Equal(t, "container1", newContainers[0].Name)
		assert.Equal(t, "container3", newContainers[1].Name)
	})

	t.Run("applies multiple filters correctly", func(t *testing.T) {
		req := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationContainerInjection + "/container2": "false",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container1"},
						{Name: "container2"},
						{Name: "container3"},
						{Name: "container4"},
					},
				},
			},
			DynaKube: dynakube.DynaKube{},
		}

		isInjected := func(c corev1.Container, _ *BaseRequest) bool {
			return c.Name == "container3"
		}

		newContainers := req.NewContainers(isInjected)
		require.Len(t, newContainers, 2)
		assert.Equal(t, "container1", newContainers[0].Name)
		assert.Equal(t, "container4", newContainers[1].Name)
	})
}

func TestToReinvocationRequest(t *testing.T) {
	t.Run("converts MutationRequest to ReinvocationRequest", func(t *testing.T) {
		baseReq := &BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
			},
			DynaKube: dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-dk",
				},
			},
			Namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
		}

		mutationReq := &MutationRequest{
			BaseRequest: baseReq,
			Context:     context.Background(),
			InstallContainer: &corev1.Container{
				Name: "installer",
			},
		}

		reinvocationReq := mutationReq.ToReinvocationRequest()

		require.NotNil(t, reinvocationReq)
		assert.Equal(t, baseReq, reinvocationReq.BaseRequest)
		assert.Equal(t, "test-pod", reinvocationReq.Pod.Name)
		assert.Equal(t, "test-dk", reinvocationReq.DynaKube.Name)
		assert.Equal(t, "test-ns", reinvocationReq.Namespace.Name)
	})
}

func TestNewMutationRequest(t *testing.T) {
	t.Run("creates mutation request with all fields", func(t *testing.T) {
		ctx := context.Background()
		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ns",
			},
		}
		installContainer := &corev1.Container{
			Name: "installer",
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
		}
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-dk",
			},
		}

		req := NewMutationRequest(ctx, namespace, installContainer, pod, dk)

		require.NotNil(t, req)
		assert.NotNil(t, req.BaseRequest)
		assert.Equal(t, ctx, req.Context)
		assert.Equal(t, installContainer, req.InstallContainer)
		assert.Equal(t, pod, req.Pod)
		assert.Equal(t, "test-dk", req.DynaKube.Name)
		assert.Equal(t, "test-ns", req.Namespace.Name)
	})
}
