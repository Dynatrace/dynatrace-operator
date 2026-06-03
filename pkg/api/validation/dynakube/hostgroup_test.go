package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInvalidOneAgentHostGroup(t *testing.T) {
	t.Run("empty host group is allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL},
		})
	})

	t.Run("valid host group property is allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:   testAPIURL,
				OneAgent: oneagent.Spec{HostGroup: "host-group"},
			},
		})
	})

	t.Run("host group property with newline is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupProperty}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL:   testAPIURL,
				OneAgent: oneagent.Spec{HostGroup: "host\ngroup"},
			},
		})
	})

	t.Run("host group property with tab is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupProperty}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL:   testAPIURL,
				OneAgent: oneagent.Spec{HostGroup: "host\tgroup"},
			},
		})
	})

	t.Run("host group property with carriage return is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupProperty}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL:   testAPIURL,
				OneAgent: oneagent.Spec{HostGroup: "host\rgroup"},
			},
		})
	})

	t.Run("host group property with null byte is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupProperty}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL:   testAPIURL,
				OneAgent: oneagent.Spec{HostGroup: "host\x00group"},
			},
		})
	})

	t.Run("valid --set-host-group arg is allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{Args: []string{"--set-host-group=host-group"}},
				},
			},
		})
	})

	t.Run("--set-host-group arg with newline is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupAsParam}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{Args: []string{"--set-host-group=host\ngroup"}},
				},
			},
		})
	})

	t.Run("--set-host-group arg with tab is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupAsParam}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{Args: []string{"--set-host-group=host\tgroup"}},
				},
			},
		})
	})

	t.Run("--set-host-group arg with carriage return is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupAsParam}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{Args: []string{"--set-host-group=host\rgroup"}},
				},
			},
		})
	})

	t.Run("--set-host-group arg with null byte is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidHostGroupAsParam}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{Args: []string{"--set-host-group=host\x00group"}},
				},
			},
		})
	})
}
