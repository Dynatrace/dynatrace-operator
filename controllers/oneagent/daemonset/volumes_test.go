package daemonset

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareVolumes(t *testing.T) {
	t.Run(`has root volume`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
		assert.NotContains(t, volumes, getReadOnlyVolume(instance))
	})
	t.Run(`has certificate volume`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				TrustedCAs: testName,
			},
		}
		volumes := prepareVolumes(instance)

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
		assert.NotContains(t, volumes, getReadOnlyVolume(instance))
	})
	t.Run(`has readonly installation volume`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				OneAgent: dynatracev1.OneAgentSpec{
					HostMonitoring: &dynatracev1.HostMonitoringSpec{
						ReadOnly: true,
					},
				},
			},
		}
		dsInfo := HostMonitoring{
			builderInfo{
				instance:       instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				logger:         logger.NewDTLogger(),
				clusterId:      "",
				relatedImage:   "",
			},
			HostMonitoringFeature,
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumes := ds.Spec.Template.Spec.Volumes

		assert.Contains(t, volumes, getRootVolume())
		assert.NotContains(t, volumes, getCertificateVolume(instance))
		assert.Contains(t, volumes, getReadOnlyVolume(instance))
	})
	t.Run(`has all volumes`, func(t *testing.T) {
		instance := &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				TrustedCAs: testName,
				OneAgent: dynatracev1.OneAgentSpec{
					HostMonitoring: &dynatracev1.HostMonitoringSpec{
						ReadOnly: true,
					},
				},
			},
		}
		dsInfo := HostMonitoring{
			builderInfo{
				instance:       instance,
				hostInjectSpec: &instance.Spec.OneAgent.HostMonitoring.HostInjectSpec,
				logger:         logger.NewDTLogger(),
				clusterId:      "",
				relatedImage:   "",
			},
			HostMonitoringFeature,
		}
		ds, err := dsInfo.BuildDaemonSet()
		require.NoError(t, err)

		volumes := ds.Spec.Template.Spec.Volumes

		assert.Contains(t, volumes, getRootVolume())
		assert.Contains(t, volumes, getCertificateVolume(instance))
		assert.Contains(t, volumes, getReadOnlyVolume(instance))
	})
}
