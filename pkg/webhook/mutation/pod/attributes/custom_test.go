package attributes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetNamespaceAnnotationAttributes(t *testing.T) {
	t.Run("stores keys with metadata prefix, stripping the prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					metadataenrichment.Prefix + "my.attr": "value1",
					metadataenrichment.Prefix + "other":   "value2",
				},
			},
		}

		attrs.getNamespaceAnnotationAttributes(ns)

		assert.Equal(t, "value1", attrs.namespaceAnnotations["my.attr"])
		assert.Equal(t, "value2", attrs.namespaceAnnotations["other"])
	})

	t.Run("ignores keys without the metadata prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"unrelated.annotation/key": "ignored",
					metadataenrichment.Prefix + "kept":  "kept-value",
				},
			},
		}

		attrs.getNamespaceAnnotationAttributes(ns)

		assert.Len(t, attrs.namespaceAnnotations, 1)
		assert.Equal(t, "kept-value", attrs.namespaceAnnotations["kept"])
	})

	t.Run("empty annotations map results in empty namespaceAnnotations", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.getNamespaceAnnotationAttributes(corev1.Namespace{})
		assert.Empty(t, attrs.namespaceAnnotations)
	})
}

func TestGetPodAnnotationAttributes(t *testing.T) {
	t.Run("stores keys with metadata prefix, stripping the prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					metadataenrichment.Prefix + "my.attr": "pod-value",
				},
			},
		}

		attrs.getPodAnnotationAttributes(pod)

		assert.Equal(t, "pod-value", attrs.podAnnotations["my.attr"])
	})

	t.Run("ignores keys without the metadata prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"unrelated/key":                      "ignored",
					metadataenrichment.Prefix + "kept":  "kept-value",
				},
			},
		}

		attrs.getPodAnnotationAttributes(pod)

		assert.Len(t, attrs.podAnnotations, 1)
	})

	t.Run("empty annotations map results in empty podAnnotations", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.getPodAnnotationAttributes(corev1.Pod{})
		assert.Empty(t, attrs.podAnnotations)
	})
}

func TestGetFromEnrichmentRules(t *testing.T) {
	t.Run("LabelRule without target stores under computed rules key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"env": "production"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env"},
					},
				},
			},
		}

		attrs.getFromEnrichmentRules(ns, dk)

		expectedKey := metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "env")
		assert.Equal(t, "production", attrs.rules[expectedKey])
		assert.Empty(t, attrs.rulesPropagate)
	})

	t.Run("LabelRule with target stores in rulesPropagate under the target key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"env": "staging"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env", Target: "custom.env"},
					},
				},
			},
		}

		attrs.getFromEnrichmentRules(ns, dk)

		assert.Equal(t, "staging", attrs.rulesPropagate["custom.env"])
		assert.Empty(t, attrs.rules)
	})

	t.Run("AnnotationRule reads from namespace annotations", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"team": "backend"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.AnnotationRule, Source: "team", Target: "team.name"},
					},
				},
			},
		}

		attrs.getFromEnrichmentRules(ns, dk)

		assert.Equal(t, "backend", attrs.rulesPropagate["team.name"])
	})

	t.Run("rule whose source is absent from namespace is skipped", func(t *testing.T) {
		attrs := newTestPodAttributes()
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "missing-label"},
					},
				},
			},
		}

		attrs.getFromEnrichmentRules(corev1.Namespace{}, dk)

		assert.Empty(t, attrs.rules)
		assert.Empty(t, attrs.rulesPropagate)
	})

	t.Run("mix of target and no-target rules routes correctly", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"env":  "prod",
					"team": "platform",
				},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env"},
						{Type: metadataenrichment.LabelRule, Source: "team", Target: "custom.team"},
					},
				},
			},
		}

		attrs.getFromEnrichmentRules(ns, dk)

		envKey := metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "env")
		assert.Equal(t, "prod", attrs.rules[envKey])
		assert.Equal(t, "platform", attrs.rulesPropagate["custom.team"])
	})
}

func TestRemoveMetadataPrefix(t *testing.T) {
	t.Run("strips metadata prefix from prefixed keys", func(t *testing.T) {
		input := map[string]string{
			metadataenrichment.Prefix + "my.attr":    "value1",
			metadataenrichment.Prefix + "other.attr": "value2",
		}
		result := RemoveMetadataPrefix(input)
		assert.Equal(t, map[string]string{"my.attr": "value1", "other.attr": "value2"}, result)
	})

	t.Run("passes through keys without the prefix unchanged", func(t *testing.T) {
		input := map[string]string{
			"no-prefix-key": "value",
		}
		result := RemoveMetadataPrefix(input)
		assert.Equal(t, map[string]string{"no-prefix-key": "value"}, result)
	})

	t.Run("handles mix of prefixed and non-prefixed keys", func(t *testing.T) {
		input := map[string]string{
			metadataenrichment.Prefix + "attr": "prefixed",
			"plain":                             "not-prefixed",
		}
		result := RemoveMetadataPrefix(input)
		assert.Equal(t, map[string]string{"attr": "prefixed", "plain": "not-prefixed"}, result)
	})

	t.Run("empty map returns empty map", func(t *testing.T) {
		result := RemoveMetadataPrefix(map[string]string{})
		assert.Empty(t, result)
	})
}

func TestGetMetadataAnnotations(t *testing.T) {
	t.Run("collects namespace annotations, pod annotations, and enrichment rules", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{metadataenrichment.Prefix + "ns-key": "ns-val"},
				Labels:      map[string]string{"env": "prod"},
			},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{metadataenrichment.Prefix + "pod-key": "pod-val"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env", Target: "custom.env"},
					},
				},
			},
		}

		attrs.GetMetadataAnnotations(dtwebhook.BaseRequest{Pod: &pod, Namespace: ns, DynaKube: dk})

		assert.Equal(t, "ns-val", attrs.namespaceAnnotations["ns-key"])
		assert.Equal(t, "pod-val", attrs.podAnnotations["pod-key"])
		assert.Equal(t, "prod", attrs.rulesPropagate["custom.env"])
	})
}
