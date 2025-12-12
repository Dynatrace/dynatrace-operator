//go:build e2e

package helpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// The e2e framework has this in an internal package, no idea why
const (
	// LevelSetup when doing the setup phase
	LevelSetup features.Level = iota
	// LevelAssess when doing the assess phase
	LevelAssess
	// LevelTeardown when doing the teardown phase
	LevelTeardown
)

func ToFeatureFunc(envFunc env.Func, isFatal bool) features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		var err error
		ctx, err = envFunc(ctx, c)
		if isFatal {
			require.NoError(t, err)
		} else {
			assert.NoError(t, err)
		}

		return ctx
	}
}
