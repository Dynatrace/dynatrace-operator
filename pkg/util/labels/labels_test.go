package labels

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/consts"
	"github.com/stretchr/testify/assert"
)

func TestConstructors(t *testing.T) {
	appLabels := NewAppLabels(consts.TestComponent, consts.TestName, consts.TestComponentFeature, consts.TestComponentVersion)
	coreLabels := NewCoreLabels(consts.TestName, consts.TestComponent)

	expectedCoreMatchLabels := map[string]string{
		AppNameLabel:      consts.TestAppName,
		AppCreatedByLabel: consts.TestName,
		AppComponentLabel: consts.TestComponent,
	}
	expectedAppMatchLabels := map[string]string{
		AppNameLabel:      consts.TestComponent,
		AppCreatedByLabel: consts.TestName,
		AppManagedByLabel: consts.TestAppName,
	}
	expectedAppLabels := map[string]string{
		AppNameLabel:      consts.TestComponent,
		AppCreatedByLabel: consts.TestName,
		AppComponentLabel: consts.TestComponentFeature,
		AppVersionLabel:   consts.TestComponentVersion,
		AppManagedByLabel: consts.TestAppName,
	}
	expectedCoreLabels := map[string]string{
		AppNameLabel:      consts.TestAppName,
		AppCreatedByLabel: consts.TestName,
		AppComponentLabel: consts.TestComponent,
		AppVersionLabel:   consts.TestAppVersion,
	}

	t.Run("verify matchLabels for statefulsetreconciler", func(t *testing.T) {
		assert.Equal(t, expectedCoreMatchLabels, coreLabels.BuildMatchLabels())
	})
	t.Run("verify labels for statefulsetreconciler", func(t *testing.T) {
		assert.Equal(t, expectedCoreLabels, coreLabels.BuildLabels())
	})

	t.Run("verify matchLabels for app", func(t *testing.T) {
		assert.Equal(t, expectedAppMatchLabels, appLabels.BuildMatchLabels())
	})
	t.Run("verify labels for app", func(t *testing.T) {
		assert.Equal(t, expectedAppLabels, appLabels.BuildLabels())
	})
}
