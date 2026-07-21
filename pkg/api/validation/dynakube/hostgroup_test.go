// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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

	t.Run("host group with invalid characters is denied", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL:   testAPIURL,
				OneAgent: oneagent.Spec{},
			},
		}

		assertSanitizeArg(t, dk, func(dk *dynakube.DynaKube, value string) {
			dk.Spec.OneAgent.HostGroup = value
		}, errorInvalidHostGroupProperty)
	})
}
