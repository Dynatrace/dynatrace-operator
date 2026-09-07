// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttrRef(t *testing.T) {
	assert.Equal(t, `attributes["k8s.pod.name"]`, attrRef("k8s.pod.name"))
}

func TestSetIfAbsent(t *testing.T) {
	assert.Equal(t,
		`set(attributes["k8s.cluster.uid"], attributes["a"]) where attributes["k8s.cluster.uid"] == nil`,
		setIfAbsent("k8s.cluster.uid", attrRef("a")),
	)
}

func TestSetValueIfPresentAndAbsent(t *testing.T) {
	assert.Equal(t,
		`set(attributes["k8s.workload.name"], attributes["k8s.statefulset.name"]) where IsString(attributes["k8s.statefulset.name"]) and attributes["k8s.workload.name"] == nil`,
		setValueIfPresentAndAbsent("k8s.workload.name", "k8s.statefulset.name"),
	)
}

func TestSetLiteralIfAbsent(t *testing.T) {
	assert.Equal(t,
		`set(attributes["k8s.cluster.uid"], "${env:K8S_CLUSTER_UID}") where attributes["k8s.cluster.uid"] == nil`,
		setLiteralIfAbsent("k8s.cluster.uid", "${env:K8S_CLUSTER_UID}"),
	)
}

func TestSetLiteralIfPresentAndAbsent(t *testing.T) {
	assert.Equal(t,
		`set(attributes["k8s.workload.kind"], "statefulset") where IsString(attributes["k8s.statefulset.name"]) and attributes["k8s.workload.kind"] == nil`,
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "statefulset", "k8s.statefulset.name"),
	)
}

func TestDeleteKeys(t *testing.T) {
	assert.Equal(t,
		[]string{`delete_key(attributes, "a")`, `delete_key(attributes, "b")`},
		deleteKeys("a", "b"),
	)
}
