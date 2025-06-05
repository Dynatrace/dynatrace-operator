//go:build e2e

package job

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func WaitForJobsDeletionWithOwner(ownerName string, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		jobs := GetJobsForOwner(ctx, t, resources, ownerName, namespace)

		if len(jobs.Items) > 0 {
			err := wait.For(conditions.New(resources).ResourcesDeleted(&jobs), wait.WithTimeout(1*time.Minute))
			require.NoError(t, err)
		}

		return ctx
	}
}
