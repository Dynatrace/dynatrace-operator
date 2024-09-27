package validation

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/stretchr/testify/assert"
)

func TestIsModuleDisabled(t *testing.T) {
	ctx := context.Background()

	t.Run("module disabled => error", func(t *testing.T) {
		errMsg := isModuleDisabled(ctx, &Validator{modules: installconfig.Modules{EdgeConnect: false}}, nil)
		assert.Equal(t, errorModuleDisabled, errMsg)
	})

	t.Run("module enabled => no error", func(t *testing.T) {
		errMsg := isModuleDisabled(ctx, &Validator{modules: installconfig.Modules{EdgeConnect: true}}, nil)
		assert.Empty(t, errMsg)
	})
}
