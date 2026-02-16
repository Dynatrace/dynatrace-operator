//go:build e2e

package k8sjob

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func ListForOwner(ctx context.Context, t *testing.T, resource *resources.Resources, ownerName, namespace string) batchv1.JobList {
	jobs := getJobsForNamespace(ctx, t, resource, namespace)

	var targetJobs batchv1.JobList
	for _, jobItem := range jobs.Items {
		if len(jobItem.OwnerReferences) < 1 {
			continue
		}

		if jobItem.OwnerReferences[0].Name == ownerName {
			targetJobs.Items = append(targetJobs.Items, jobItem)
		}
	}

	return targetJobs
}

func getJobsForNamespace(ctx context.Context, t *testing.T, resource *resources.Resources, namespace string) batchv1.JobList {
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
