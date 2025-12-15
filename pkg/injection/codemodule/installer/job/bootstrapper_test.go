package job

import (
	"slices"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/csijob"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildJobName(t *testing.T) {
	t.Run("job names are unique", func(t *testing.T) {
		uris := []string{"test1.com", "test2.com", "test3.com", "test4.com", "test5.com"}
		nodeNames := []string{"node1", "node2", "node3", "node4", "node5"}

		jobNames := []string{}

		for i := range nodeNames {
			for j := range uris {
				props := &Properties{
					ImageURI: uris[j],
				}
				inst := &Installer{
					nodeName: nodeNames[i],
					props:    props,
				}
				jobNames = append(jobNames, inst.buildJobName())
			}
		}

		slices.Sort(jobNames)
		jobNames = slices.Compact(jobNames)

		assert.Len(t, jobNames, len(nodeNames)*len(uris))
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run("args are built correctly", func(t *testing.T) {
		targetDir := "1.2.3"
		jobName := "test-job-123"
		props := &Properties{
			PathResolver: metadata.PathResolver{RootDir: "root"},
		}
		inst := &Installer{
			props: props,
		}

		args := inst.buildArgs(jobName, targetDir)

		require.Len(t, args, 3)
		assert.Contains(t, args[0], "--source")
		assert.Contains(t, args[0], codeModuleSource)

		assert.Contains(t, args[1], "--target")
		assert.Contains(t, args[1], targetDir)

		assert.Contains(t, args[2], "--work")
		assert.Contains(t, args[2], props.PathResolver.RootDir)
		assert.Contains(t, args[2], jobName)
	})
}

func TestBuildJob(t *testing.T) {
	owner := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-dk",
		},
	}
	name := "job-1"
	imageURI := "test:5000/repo"
	nodeName := "node1"
	targetDir := "1.2.3"
	pullSecrets := []string{"secret-1", "secret-2"}

	t.Run("job created correctly", func(t *testing.T) {
		dataDir := setupDataDir(t)

		props := &Properties{
			Owner:        &owner,
			ImageURI:     imageURI,
			PullSecrets:  pullSecrets,
			PathResolver: metadata.PathResolver{RootDir: "root"},
			CSIJob:       csijob.GetSettings(),
		}
		inst := &Installer{
			nodeName: nodeName,
			props:    props,
		}

		job, err := inst.buildJob(name, targetDir)
		require.NoError(t, err)
		require.NotNil(t, job)

		// Global/pod level checks
		assert.Equal(t, name, job.Name)
		assert.NotNil(t, job.Labels)
		assert.NotNil(t, job.Spec.Template.Labels)
		assert.Empty(t, job.Spec.Selector) // the Job objects handles this by default, our generated MatchLabels wound't even work

		assert.NotEmpty(t, job.Spec.Template.Spec.Tolerations)
		assert.Equal(t, nodeName, job.Spec.Template.Spec.NodeName)
		assert.False(t, *job.Spec.Template.Spec.AutomountServiceAccountToken)
		assert.Equal(t, corev1.RestartPolicyOnFailure, job.Spec.Template.Spec.RestartPolicy)

		require.Len(t, job.Spec.Template.Spec.Volumes, 1)
		require.NotNil(t, job.Spec.Template.Spec.Volumes[0].HostPath)
		assert.Equal(t, dataDir, job.Spec.Template.Spec.Volumes[0].HostPath.Path)

		require.Equal(t, provisionerServiceAccount, job.Spec.Template.Spec.ServiceAccountName)
		require.Equal(t, provisionerServiceAccount, job.Spec.Template.Spec.DeprecatedServiceAccount)

		require.Len(t, job.Spec.Template.Spec.ImagePullSecrets, len(pullSecrets))

		for i, ps := range pullSecrets {
			assert.Equal(t, ps, job.Spec.Template.Spec.ImagePullSecrets[i].Name)
		}

		// Container level checks
		require.Len(t, job.Spec.Template.Spec.Containers, 1)
		container := job.Spec.Template.Spec.Containers[0]

		assert.Equal(t, imageURI, container.Image)
		assert.Empty(t, container.Command)
		assert.NotEmpty(t, container.Args)
		assert.NotEmpty(t, container.SecurityContext)

		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, job.Spec.Template.Spec.Volumes[0].Name, container.VolumeMounts[0].Name)
		assert.Equal(t, props.PathResolver.RootDir, container.VolumeMounts[0].MountPath)
	})

	t.Run("job uses priority class when set", func(t *testing.T) {
		priorityClassName := "dynatrace-high-priority"
		priority := int32(1000000)

		props := &Properties{
			Owner:             &owner,
			ImageURI:          imageURI,
			PullSecrets:       pullSecrets,
			PathResolver:      metadata.PathResolver{RootDir: "root"},
			CSIJob:            csijob.GetSettings(),
			PriorityClassName: priorityClassName,
			Priority:          priority,
		}
		inst := &Installer{
			nodeName: nodeName,
			props:    props,
		}

		job, err := inst.buildJob(name, targetDir)
		require.NoError(t, err)
		require.NotNil(t, job)

		assert.Equal(t, priorityClassName, job.Spec.Template.Spec.PriorityClassName)
		assert.Equal(t, priority, *job.Spec.Template.Spec.Priority)
	})
}

func setupDataDir(t *testing.T) string {
	t.Helper()

	dataDir := "test/data"

	t.Setenv(k8senv.CSIDataDir, dataDir)

	return dataDir
}
