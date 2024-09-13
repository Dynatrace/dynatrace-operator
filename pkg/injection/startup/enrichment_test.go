package startup

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEnrichmentFiles(t *testing.T) {
	t.Run("create enrichment files", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		runner := Runner{
			fs: fs,
			env: &environment{
				K8PodUID:          "K8PodUID",
				K8PodName:         "K8PodName",
				K8NodeName:        "K8NodeName",
				K8Namespace:       "K8Namespace",
				K8ClusterName:     "K8ClusterName",
				K8ClusterEntityID: "K8EntityID",
				K8ClusterID:       "K8ClusterID",
				WorkloadKind:      "WorkloadKind",
				WorkloadName:      "WorkloadName",
				Containers: []ContainerInfo{
					{
						Name: "Container-1",
					},
					{
						Name: "Container-2",
					},
				},
			},
		}

		err := runner.createEnrichmentFiles()
		require.NoError(t, err)

		for _, container := range runner.env.Containers {
			expectedJson := fmt.Sprintf("{\"dt.entity.kubernetes_cluster\":\"K8EntityID\",\"dt.kubernetes.cluster.id\":\"K8ClusterID\",\"dt.kubernetes.workload.kind\":\"WorkloadKind\",\"dt.kubernetes.workload.name\":\"WorkloadName\",\"k8s.cluster.name\":\"K8ClusterName\",\"k8s.cluster.uid\":\"K8ClusterID\",\"k8s.container.name\":\"%s\",\"k8s.namespace.name\":\"K8Namespace\",\"k8s.node.name\":\"K8NodeName\",\"k8s.pod.name\":\"K8PodName\",\"k8s.pod.uid\":\"K8PodUID\",\"k8s.workload.kind\":\"WorkloadKind\",\"k8s.workload.name\":\"WorkloadName\"}", container.Name)

			jsonFile, err := fs.Open(fmt.Sprintf(enrichmentJsonPathTemplate, container.Name))
			require.NoError(t, err)
			content, err := io.ReadAll(jsonFile)
			require.NoError(t, err)
			assert.Equal(t, expectedJson, string(content))

			expectedProps := map[string]string{}
			err = json.Unmarshal(content, &expectedProps)
			require.NoError(t, err)

			propsFile, err := fs.Open(fmt.Sprintf(enrichmentPropsPathTemplate, container.Name))
			require.NoError(t, err)
			content, err = io.ReadAll(propsFile)
			require.NoError(t, err)

			for key, value := range expectedProps {
				assert.Contains(t, string(content), key+"="+value)
			}
		}
	})
	t.Run("omit cluster name if not there", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		runner := Runner{
			fs: fs,
			env: &environment{
				K8PodUID:     "K8PodUID",
				K8PodName:    "K8PodName",
				K8NodeName:   "K8NodeName",
				K8Namespace:  "K8Namespace",
				K8ClusterID:  "K8ClusterID",
				WorkloadKind: "WorkloadKind",
				WorkloadName: "WorkloadName",
				Containers: []ContainerInfo{
					{
						Name: "Container-1",
					},
					{
						Name: "Container-2",
					},
				},
			},
		}

		err := runner.createEnrichmentFiles()
		require.NoError(t, err)

		for _, container := range runner.env.Containers {
			jsonFile, err := fs.Open(fmt.Sprintf(enrichmentJsonPathTemplate, container.Name))
			require.NoError(t, err)
			content, err := io.ReadAll(jsonFile)
			require.NoError(t, err)
			assert.NotContains(t, string(content), "k8s.cluster.name")
			assert.NotContains(t, string(content), "dt.entity.kubernetes_cluster")

			propsFile, err := fs.Open(fmt.Sprintf(enrichmentPropsPathTemplate, container.Name))
			require.NoError(t, err)
			content, err = io.ReadAll(propsFile)
			require.NoError(t, err)
			assert.NotContains(t, string(content), "k8s.cluster.name=")
			assert.NotContains(t, string(content), "dt.entity.kubernetes_cluster=")
		}
	})

	t.Run("add properties from DT_WORKLOAD_ANNOTATIONS", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		runner := Runner{
			fs: fs,
			env: &environment{
				K8PodUID:          "K8PodUID",
				K8PodName:         "K8PodName",
				K8NodeName:        "K8NodeName",
				K8Namespace:       "K8Namespace",
				K8ClusterName:     "K8ClusterName",
				K8ClusterEntityID: "K8EntityID",
				K8ClusterID:       "K8ClusterID",
				WorkloadKind:      "WorkloadKind",
				WorkloadName:      "WorkloadName",
				WorkloadAnnotations: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Containers: []ContainerInfo{
					{
						Name: "Container-1",
					},
					{
						Name: "Container-2",
					},
				},
			},
		}

		err := runner.createEnrichmentFiles()
		require.NoError(t, err)

		for _, container := range runner.env.Containers {
			expectedJson := fmt.Sprintf("{\"dt.entity.kubernetes_cluster\":\"K8EntityID\",\"dt.kubernetes.cluster.id\":\"K8ClusterID\",\"dt.kubernetes.workload.kind\":\"WorkloadKind\",\"dt.kubernetes.workload.name\":\"WorkloadName\",\"k8s.cluster.name\":\"K8ClusterName\",\"k8s.cluster.uid\":\"K8ClusterID\",\"k8s.container.name\":\"%s\",\"k8s.namespace.name\":\"K8Namespace\",\"k8s.node.name\":\"K8NodeName\",\"k8s.pod.name\":\"K8PodName\",\"k8s.pod.uid\":\"K8PodUID\",\"k8s.workload.kind\":\"WorkloadKind\",\"k8s.workload.name\":\"WorkloadName\",\"key1\":\"value1\",\"key2\":\"value2\"}", container.Name)

			jsonFile, err := fs.Open(fmt.Sprintf(enrichmentJsonPathTemplate, container.Name))
			require.NoError(t, err)
			content, err := io.ReadAll(jsonFile)
			require.NoError(t, err)
			assert.Equal(t, expectedJson, string(content))

			expectedProps := map[string]string{}
			err = json.Unmarshal(content, &expectedProps)
			require.NoError(t, err)

			propsFile, err := fs.Open(fmt.Sprintf(enrichmentPropsPathTemplate, container.Name))
			require.NoError(t, err)
			content, err = io.ReadAll(propsFile)
			require.NoError(t, err)

			for key, value := range expectedProps {
				assert.Contains(t, string(content), key+"="+value)
			}
		}
	})
}
