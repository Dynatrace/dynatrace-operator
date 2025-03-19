//go:build e2e

package job

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func GetJobsForOwner(ctx context.Context, t *testing.T, resource *resources.Resources, ownerName, namespace string) batchv1.JobList {
	jobs := GetJobsForNamespace(ctx, t, resource, namespace)

	var targetJobs batchv1.JobList
	for _, pod := range jobs.Items {
		if len(pod.ObjectMeta.OwnerReferences) < 1 {
			continue
		}

		if pod.ObjectMeta.OwnerReferences[0].Name == ownerName {
			targetJobs.Items = append(targetJobs.Items, pod)
		}
	}

	return targetJobs
}

func GetJobsForNamespace(ctx context.Context, t *testing.T, resource *resources.Resources, namespace string) batchv1.JobList {
	var jobs batchv1.JobList
	err := resource.WithNamespace(namespace).List(ctx, &jobs)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = nil
		}
		require.NoError(t, err)
	}

	return jobs
}
