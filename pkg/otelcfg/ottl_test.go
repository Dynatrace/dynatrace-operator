// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package otelcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteKeys(t *testing.T) {
	assert.Equal(
		t,
		[]string{`delete_key(attributes, "a")`, `delete_key(attributes, "b")`},
		DeleteKeys("a", "b"),
	)
}

func TestRenameAttr(t *testing.T) {
	assert.Equal(
		t,
		[]string{`set(attributes["a"], attributes["b"]) where attributes["b"] != nil`, `delete_key(attributes, "b")`},
		RenameAttr("a", "b"),
	)
}

func TestSetAttributeFromEnv(t *testing.T) {
	assert.Equal(t, `set(attributes["a"], "${env:b}") where attributes["a"] == nil and Len("${env:b}") > 0`, SetAttrFromEnv("a", "b"))
}

func TestAttrRef(t *testing.T) {
	assert.Equal(t, `attributes["k8s.pod.name"]`, attrRef("k8s.pod.name"))
}

func TestSetAttrIfAbsent(t *testing.T) {
	assert.Equal(
		t,
		`set(attributes["k8s.cluster.uid"], attributes["a"]) where attributes["k8s.cluster.uid"] == nil`,
		SetAttrIfAbsent("k8s.cluster.uid", "a"),
	)
}

func TestSetValueIfPresentAndAbsent(t *testing.T) {
	assert.Equal(
		t,
		`set(attributes["k8s.workload.name"], attributes["k8s.statefulset.name"]) where IsString(attributes["k8s.statefulset.name"]) and attributes["k8s.workload.name"] == nil`,
		setValueIfPresentAndAbsent("k8s.workload.name", "k8s.statefulset.name"),
	)
}

func TestSetLiteralIfPresentAndAbsent(t *testing.T) {
	assert.Equal(
		t,
		`set(attributes["k8s.workload.kind"], "statefulset") where IsString(attributes["k8s.statefulset.name"]) and attributes["k8s.workload.kind"] == nil`,
		setLiteralIfPresentAndAbsent("k8s.workload.kind", "statefulset", "k8s.statefulset.name"),
	)
}
