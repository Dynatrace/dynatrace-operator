package supportarchive

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const nodeTaintAnalysisCollectorName = "nodeTaintAnalysisCollector"

type nodeTaintAnalysisCollector struct {
	collectorCommon
	ctx       context.Context
	apiReader client.Reader
	namespace string
}

func newNodeTaintAnalysisCollector(ctx context.Context, log logd.Logger, supportArchive archiver, namespace string, apiReader client.Reader) collector {
	return nodeTaintAnalysisCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		ctx:       ctx,
		apiReader: apiReader,
		namespace: namespace,
	}
}

func (c nodeTaintAnalysisCollector) Name() string {
	return nodeTaintAnalysisCollectorName
}

func (c nodeTaintAnalysisCollector) Do() error {
	logInfof(c.log, "Starting node taint analysis")

	report, err := c.buildReport()
	if err != nil {
		logErrorf(c.log, err, "Failed to complete node taint analysis")

		return err
	}

	if err := c.supportArchive.addFile(NodeTaintAnalysisFileName, strings.NewReader(report)); err != nil {
		return err
	}

	logInfof(c.log, "Stored node taint analysis into %s", NodeTaintAnalysisFileName)

	return nil
}

func (c nodeTaintAnalysisCollector) buildReport() (string, error) {
	nodes, err := c.getNodes()
	if err != nil {
		return "", err
	}

	dynakubes, err := c.getDynaKubes()
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Node count: %d\n\n", len(nodes.Items)))

	c.analyzeOneAgentDaemonSets(&sb, nodes, dynakubes)
	c.analyzeCSIDaemonSet(&sb, nodes)

	return sb.String(), nil
}

func (c nodeTaintAnalysisCollector) analyzeOneAgentDaemonSets(sb *strings.Builder, nodes *corev1.NodeList, dynakubes *dynakube.DynaKubeList) {
	if len(dynakubes.Items) == 0 {
		sb.WriteString("No DynaKube resources found, skipping OneAgent DaemonSet analysis\n")

		return
	}

	for i := range dynakubes.Items {
		dk := &dynakubes.Items[i]
		dsName := dk.OneAgent().GetDaemonsetName()

		sb.WriteString(fmt.Sprintf("--- OneAgent DaemonSet: %s (DynaKube: %s) ---\n", dsName, dk.Name))

		var ds appsv1.DaemonSet

		err := c.apiReader.Get(c.ctx, client.ObjectKey{Name: dsName, Namespace: c.namespace}, &ds)
		if err != nil {
			sb.WriteString(fmt.Sprintf("  DaemonSet not found: %s\n\n", err.Error()))

			continue
		}

		c.analyzeDaemonSet(sb, &ds, nodes)
		sb.WriteString("\n")
	}
}

func (c nodeTaintAnalysisCollector) analyzeCSIDaemonSet(sb *strings.Builder, nodes *corev1.NodeList) {
	sb.WriteString(fmt.Sprintf("--- CSI Driver DaemonSet: %s ---\n", dtcsi.DaemonSetName))

	var ds appsv1.DaemonSet

	err := c.apiReader.Get(c.ctx, client.ObjectKey{Name: dtcsi.DaemonSetName, Namespace: c.namespace}, &ds)
	if err != nil {
		sb.WriteString(fmt.Sprintf("  DaemonSet not found: %s\n\n", err.Error()))

		return
	}

	c.analyzeDaemonSet(sb, &ds, nodes)
	sb.WriteString("\n")
}

func (c nodeTaintAnalysisCollector) analyzeDaemonSet(sb *strings.Builder, ds *appsv1.DaemonSet, nodes *corev1.NodeList) {
	desired := ds.Status.DesiredNumberScheduled
	ready := ds.Status.NumberReady

	sb.WriteString(fmt.Sprintf("  Desired: %d | Ready: %d | Nodes: %d\n", desired, ready, len(nodes.Items)))

	tolerations := ds.Spec.Template.Spec.Tolerations
	sb.WriteString("  Configured tolerations:\n")

	if len(tolerations) == 0 {
		sb.WriteString("    (none)\n")
	}

	for _, t := range tolerations {
		sb.WriteString(fmt.Sprintf("    %s\n", formatToleration(t)))
	}

	untoleratedNodes := findNodesWithUntoleratedTaints(nodes, tolerations)
	if len(untoleratedNodes) == 0 {
		sb.WriteString("  All node taints are tolerated\n")

		return
	}

	sb.WriteString(fmt.Sprintf("  WARNING: %d node(s) have untolerated taints:\n", len(untoleratedNodes)))

	for _, entry := range untoleratedNodes {
		sb.WriteString(fmt.Sprintf("    Node: %s\n", entry.nodeName))

		for _, taint := range entry.untoleratedTaints {
			sb.WriteString(fmt.Sprintf("      - %s\n", formatTaint(taint)))
		}
	}
}

type nodeWithUntoleratedTaints struct {
	nodeName         string
	untoleratedTaints []corev1.Taint
}

func findNodesWithUntoleratedTaints(nodes *corev1.NodeList, tolerations []corev1.Toleration) []nodeWithUntoleratedTaints {
	var result []nodeWithUntoleratedTaints

	for i := range nodes.Items {
		node := &nodes.Items[i]
		if len(node.Spec.Taints) == 0 {
			continue
		}

		var untolerated []corev1.Taint

		for _, taint := range node.Spec.Taints {
			if !isTaintTolerated(taint, tolerations) {
				untolerated = append(untolerated, taint)
			}
		}

		if len(untolerated) > 0 {
			result = append(result, nodeWithUntoleratedTaints{
				nodeName:         node.Name,
				untoleratedTaints: untolerated,
			})
		}
	}

	return result
}

func isTaintTolerated(taint corev1.Taint, tolerations []corev1.Toleration) bool {
	for _, toleration := range tolerations {
		if tolerationMatchesTaint(toleration, taint) {
			return true
		}
	}

	return false
}

func tolerationMatchesTaint(toleration corev1.Toleration, taint corev1.Taint) bool {
	// Empty key with Exists operator matches all taints
	if toleration.Key == "" && toleration.Operator == corev1.TolerationOpExists {
		if toleration.Effect == "" || toleration.Effect == taint.Effect {
			return true
		}

		return false
	}

	if toleration.Key != taint.Key {
		return false
	}

	if toleration.Effect != "" && toleration.Effect != taint.Effect {
		return false
	}

	if toleration.Operator == corev1.TolerationOpExists {
		return true
	}

	return toleration.Value == taint.Value
}

func formatToleration(t corev1.Toleration) string {
	parts := []string{}

	if t.Key == "" {
		parts = append(parts, "key=*")
	} else {
		parts = append(parts, fmt.Sprintf("key=%s", t.Key))
	}

	parts = append(parts, fmt.Sprintf("operator=%s", t.Operator))

	if t.Value != "" {
		parts = append(parts, fmt.Sprintf("value=%s", t.Value))
	}

	if t.Effect != "" {
		parts = append(parts, fmt.Sprintf("effect=%s", t.Effect))
	}

	return strings.Join(parts, " ")
}

func formatTaint(t corev1.Taint) string {
	if t.Value != "" {
		return fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect)
	}

	return fmt.Sprintf("%s:%s", t.Key, t.Effect)
}

func (c nodeTaintAnalysisCollector) getNodes() (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	if err := c.apiReader.List(c.ctx, nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (c nodeTaintAnalysisCollector) getDynaKubes() (*dynakube.DynaKubeList, error) {
	dkList := &dynakube.DynaKubeList{}
	if err := c.apiReader.List(c.ctx, dkList, client.InNamespace(c.namespace)); err != nil {
		return nil, err
	}

	return dkList, nil
}
