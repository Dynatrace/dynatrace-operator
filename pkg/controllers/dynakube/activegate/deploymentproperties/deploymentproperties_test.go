// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package deploymentproperties

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildContent(t *testing.T) {
	t.Run("nil map produces empty string", func(t *testing.T) {
		content := BuildContent(&dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: nil,
			},
		})
		assert.Empty(t, content)
	})

	t.Run("empty map produces empty string", func(t *testing.T) {
		content := BuildContent(&dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{},
			},
		})
		assert.Empty(t, content)
	})

	t.Run("single entry", func(t *testing.T) {
		content := BuildContent(&dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{"foo": "bar"},
			},
		})
		assert.Equal(t, "[resource_attributes]\nfoo = bar\n\n", content)
	})

	t.Run("multiple entries are sorted by key", func(t *testing.T) {
		attrs := map[string]string{
			"zzz": "last",
			"aaa": "first",
			"mmm": "middle",
		}
		content := BuildContent(&dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: attrs,
			},
		})
		assert.Equal(t, "[resource_attributes]\naaa = first\nmmm = middle\nzzz = last\n\n", content)
	})

	t.Run("no-proxy value", func(t *testing.T) {
		content := BuildContent(&dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.NoProxyKey: "1.2.3.4",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{
					Value: "1.1.1.1",
				},
			},
		})
		assert.Equal(t, "[http.client.internal]\nproxy-non-proxy-hosts = 1.2.3.4\n\n", content)
	})

	t.Run("resource attributes and no-proxy", func(t *testing.T) {
		content := BuildContent(&dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.NoProxyKey: "1.2.3.4",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{
					"foo": "bar",
				},
				Proxy: &value.Source{
					Value: "1.1.1.1",
				},
			},
		})
		assert.Equal(t, "[resource_attributes]\nfoo = bar\n\n[http.client.internal]\nproxy-non-proxy-hosts = 1.2.3.4\n\n", content)
	})
}
