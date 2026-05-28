package supportarchive

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNodeTaintAnalysisCollector_Name(t *testing.T) {
	logBuffer := bytes.Buffer{}
	c := newNodeTaintAnalysisCollector(context.Background(), newSupportArchiveLogger(&logBuffer), nil, testOperatorNamespace, nil)
	assert.Equal(t, nodeTaintAnalysisCollectorName, c.Name())
}

func TestNodeTaintAnalysisCollector_AllTaintsTolerated(t *testing.T) {
	nodes := []corev1.Node{
		createNode("node-1", corev1.Taint{Key: "key1", Value: "val1", Effect: corev1.TaintEffectNoSchedule}),
		createNode("node-2"),
	}

	dk := createTestDynaKube("my-dynakube")
	oaDS := createDaemonSet("my-dynakube-oneagent", testOperatorNamespace, 2, 2,
		corev1.Toleration{Key: "key1", Operator: corev1.TolerationOpEqual, Value: "val1", Effect: corev1.TaintEffectNoSchedule},
	)
	csiDS := createDaemonSet(dtcsi.DaemonSetName, testOperatorNamespace, 2, 2,
		corev1.Toleration{Key: "", Operator: corev1.TolerationOpExists},
	)

	content := runNodeTaintAnalysis(t, nodes, []dynakube.DynaKube{dk}, []appsv1.DaemonSet{oaDS, csiDS})

	assert.Contains(t, content, "Node count: 2")
	assert.Contains(t, content, "OneAgent DaemonSet: my-dynakube-oneagent")
	assert.Contains(t, content, "Desired: 2 | Ready: 2 | Nodes: 2")
	assert.Contains(t, content, "All node taints are tolerated")
	assert.NotContains(t, content, "WARNING")
}

func TestNodeTaintAnalysisCollector_UntoleratedTaints(t *testing.T) {
	nodes := []corev1.Node{
		createNode("node-ok"),
		createNode("node-tainted",
			corev1.Taint{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule},
			corev1.Taint{Key: "nvidia.com/gpu", Effect: corev1.TaintEffectNoSchedule},
		),
	}

	dk := createTestDynaKube("prod-dynakube")
	oaDS := createDaemonSet("prod-dynakube-oneagent", testOperatorNamespace, 1, 1)
	csiDS := createDaemonSet(dtcsi.DaemonSetName, testOperatorNamespace, 1, 1)

	content := runNodeTaintAnalysis(t, nodes, []dynakube.DynaKube{dk}, []appsv1.DaemonSet{oaDS, csiDS})

	assert.Contains(t, content, "Node count: 2")
	assert.Contains(t, content, "WARNING: 1 node(s) have untolerated taints")
	assert.Contains(t, content, "Node: node-tainted")
	assert.Contains(t, content, "dedicated=gpu:NoSchedule")
	assert.Contains(t, content, "nvidia.com/gpu:NoSchedule")
}

func TestNodeTaintAnalysisCollector_NoDynaKubes(t *testing.T) {
	nodes := []corev1.Node{
		createNode("node-1"),
	}
	csiDS := createDaemonSet(dtcsi.DaemonSetName, testOperatorNamespace, 1, 1)

	content := runNodeTaintAnalysis(t, nodes, nil, []appsv1.DaemonSet{csiDS})

	assert.Contains(t, content, "No DynaKube resources found")
	assert.Contains(t, content, "CSI Driver DaemonSet")
}

func TestNodeTaintAnalysisCollector_MissingDaemonSets(t *testing.T) {
	nodes := []corev1.Node{
		createNode("node-1"),
	}
	dk := createTestDynaKube("my-dk")

	content := runNodeTaintAnalysis(t, nodes, []dynakube.DynaKube{dk}, nil)

	assert.Contains(t, content, "DaemonSet not found")
}

func TestNodeTaintAnalysisCollector_WildcardToleration(t *testing.T) {
	nodes := []corev1.Node{
		createNode("node-1",
			corev1.Taint{Key: "anything", Value: "whatever", Effect: corev1.TaintEffectNoExecute},
			corev1.Taint{Key: "other", Effect: corev1.TaintEffectNoSchedule},
		),
	}

	dk := createTestDynaKube("dk")
	oaDS := createDaemonSet("dk-oneagent", testOperatorNamespace, 1, 1,
		corev1.Toleration{Operator: corev1.TolerationOpExists},
	)
	csiDS := createDaemonSet(dtcsi.DaemonSetName, testOperatorNamespace, 1, 1,
		corev1.Toleration{Operator: corev1.TolerationOpExists},
	)

	content := runNodeTaintAnalysis(t, nodes, []dynakube.DynaKube{dk}, []appsv1.DaemonSet{oaDS, csiDS})

	assert.Contains(t, content, "All node taints are tolerated")
	assert.NotContains(t, content, "WARNING")
}

func TestNodeTaintAnalysisCollector_PartialToleration(t *testing.T) {
	nodes := []corev1.Node{
		createNode("node-1",
			corev1.Taint{Key: "key1", Value: "val1", Effect: corev1.TaintEffectNoSchedule},
			corev1.Taint{Key: "key2", Value: "val2", Effect: corev1.TaintEffectNoExecute},
		),
	}

	dk := createTestDynaKube("dk")
	// Only tolerates key1, not key2
	oaDS := createDaemonSet("dk-oneagent", testOperatorNamespace, 0, 0,
		corev1.Toleration{Key: "key1", Operator: corev1.TolerationOpEqual, Value: "val1", Effect: corev1.TaintEffectNoSchedule},
	)

	content := runNodeTaintAnalysis(t, nodes, []dynakube.DynaKube{dk}, []appsv1.DaemonSet{oaDS})

	assert.Contains(t, content, "WARNING: 1 node(s) have untolerated taints")
	assert.Contains(t, content, "Node: node-1")
	assert.Contains(t, content, "key2=val2:NoExecute")
	assert.NotContains(t, content, "key1=val1:NoSchedule")
}

func TestTolerationMatching(t *testing.T) {
	tests := []struct {
		name       string
		taint      corev1.Taint
		toleration corev1.Toleration
		matches    bool
	}{
		{
			name:       "exact match",
			taint:      corev1.Taint{Key: "k", Value: "v", Effect: corev1.TaintEffectNoSchedule},
			toleration: corev1.Toleration{Key: "k", Operator: corev1.TolerationOpEqual, Value: "v", Effect: corev1.TaintEffectNoSchedule},
			matches:    true,
		},
		{
			name:       "exists operator matches any value",
			taint:      corev1.Taint{Key: "k", Value: "v", Effect: corev1.TaintEffectNoSchedule},
			toleration: corev1.Toleration{Key: "k", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
			matches:    true,
		},
		{
			name:       "wildcard matches everything",
			taint:      corev1.Taint{Key: "anything", Value: "val", Effect: corev1.TaintEffectNoExecute},
			toleration: corev1.Toleration{Operator: corev1.TolerationOpExists},
			matches:    true,
		},
		{
			name:       "wrong key",
			taint:      corev1.Taint{Key: "k1", Value: "v", Effect: corev1.TaintEffectNoSchedule},
			toleration: corev1.Toleration{Key: "k2", Operator: corev1.TolerationOpEqual, Value: "v", Effect: corev1.TaintEffectNoSchedule},
			matches:    false,
		},
		{
			name:       "wrong effect",
			taint:      corev1.Taint{Key: "k", Value: "v", Effect: corev1.TaintEffectNoExecute},
			toleration: corev1.Toleration{Key: "k", Operator: corev1.TolerationOpEqual, Value: "v", Effect: corev1.TaintEffectNoSchedule},
			matches:    false,
		},
		{
			name:       "empty effect tolerates all effects",
			taint:      corev1.Taint{Key: "k", Value: "v", Effect: corev1.TaintEffectNoExecute},
			toleration: corev1.Toleration{Key: "k", Operator: corev1.TolerationOpEqual, Value: "v"},
			matches:    true,
		},
		{
			name:       "wrong value",
			taint:      corev1.Taint{Key: "k", Value: "v1", Effect: corev1.TaintEffectNoSchedule},
			toleration: corev1.Toleration{Key: "k", Operator: corev1.TolerationOpEqual, Value: "v2", Effect: corev1.TaintEffectNoSchedule},
			matches:    false,
		},
		{
			name:       "wildcard with effect only matches that effect",
			taint:      corev1.Taint{Key: "k", Value: "v", Effect: corev1.TaintEffectNoExecute},
			toleration: corev1.Toleration{Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
			matches:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.matches, tolerationMatchesTaint(tt.toleration, tt.taint))
		})
	}
}

func runNodeTaintAnalysis(t *testing.T, nodes []corev1.Node, dks []dynakube.DynaKube, daemonSets []appsv1.DaemonSet) string {
	t.Helper()

	var objects []client.Object

	for i := range nodes {
		objects = append(objects, &nodes[i])
	}

	for i := range dks {
		objects = append(objects, &dks[i])
	}

	for i := range daemonSets {
		objects = append(objects, &daemonSets[i])
	}

	clt := fake.NewClientWithIndex(objects...)

	logBuffer := bytes.Buffer{}
	buffer := bytes.Buffer{}
	archive := newZipArchive(bufio.NewWriter(&buffer))

	collector := newNodeTaintAnalysisCollector(context.Background(), newSupportArchiveLogger(&logBuffer), archive, testOperatorNamespace, clt)
	require.NoError(t, collector.Do())
	require.NoError(t, archive.Close())

	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	require.NoError(t, err)
	require.Len(t, zipReader.File, 1)
	assert.Equal(t, NodeTaintAnalysisFileName, zipReader.File[0].Name)

	reader, err := zipReader.File[0].Open()
	require.NoError(t, err)
	defer assertNoErrorOnClose(t, reader)

	content, err := io.ReadAll(reader)
	require.NoError(t, err)

	return string(content)
}

func createNode(name string, taints ...corev1.Taint) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NodeSpec{
			Taints: taints,
		},
	}
}

func createTestDynaKube(name string) dynakube.DynaKube {
	return dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testOperatorNamespace,
		},
	}
}

func createDaemonSet(name, namespace string, desired, ready int32, tolerations ...corev1.Toleration) appsv1.DaemonSet {
	return appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Tolerations: tolerations,
					Containers: []corev1.Container{
						{Name: "test", Image: "test:latest"},
					},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: desired,
			NumberReady:            ready,
		},
	}
}
