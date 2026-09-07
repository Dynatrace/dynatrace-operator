// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
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

	fmt.Fprintf(&sb, "Node count: %d\n\n", len(nodes.Items))

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
		if !dk.OneAgent().IsDaemonsetRequired() {
			continue
		}

		dsName := dk.OneAgent().GetDaemonsetName()

		fmt.Fprintf(sb, "--- OneAgent DaemonSet: %s (DynaKube: %s) ---\n", dsName, dk.Name)

		var ds appsv1.DaemonSet

		err := c.apiReader.Get(c.ctx, client.ObjectKey{Name: dsName, Namespace: c.namespace}, &ds)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				sb.WriteString("  DaemonSet not found\n\n")
			} else {
				fmt.Fprintf(sb, "  Error fetching DaemonSet: %s\n\n", err.Error())
			}

			continue
		}

		c.analyzeDaemonSet(sb, &ds, nodes)
		sb.WriteString("\n")
	}
}

func (c nodeTaintAnalysisCollector) analyzeCSIDaemonSet(sb *strings.Builder, nodes *corev1.NodeList) {
	fmt.Fprintf(sb, "--- CSI Driver DaemonSet: %s ---\n", dtcsi.DaemonSetName)

	var ds appsv1.DaemonSet

	err := c.apiReader.Get(c.ctx, client.ObjectKey{Name: dtcsi.DaemonSetName, Namespace: c.namespace}, &ds)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			sb.WriteString("  DaemonSet not found\n\n")
		} else {
			fmt.Fprintf(sb, "  Error fetching DaemonSet: %s\n\n", err.Error())
		}

		return
	}

	c.analyzeDaemonSet(sb, &ds, nodes)
	sb.WriteString("\n")
}

func (c nodeTaintAnalysisCollector) analyzeDaemonSet(sb *strings.Builder, ds *appsv1.DaemonSet, nodes *corev1.NodeList) {
	desired := ds.Status.DesiredNumberScheduled
	ready := ds.Status.NumberReady

	fmt.Fprintf(sb, "  Desired: %d | Ready: %d | Total cluster nodes: %d\n", desired, ready, len(nodes.Items))

	tolerations := ds.Spec.Template.Spec.Tolerations

	sb.WriteString("  Configured tolerations:\n")

	if len(tolerations) == 0 {
		sb.WriteString("    (none)\n")
	}

	for _, t := range tolerations {
		fmt.Fprintf(sb, "    %s\n", formatToleration(t))
	}

	untoleratedNodes := findNodesWithUntoleratedTaints(nodes, tolerations)
	if len(untoleratedNodes) == 0 {
		sb.WriteString("  All node taints are tolerated\n")

		return
	}

	fmt.Fprintf(sb, "  WARNING: %d node(s) have untolerated taints:\n", len(untoleratedNodes))

	for _, entry := range untoleratedNodes {
		fmt.Fprintf(sb, "    Node: %s\n", entry.nodeName)

		for _, taint := range entry.untoleratedTaints {
			fmt.Fprintf(sb, "      - %s\n", formatTaint(taint))
		}
	}
}

type nodeWithUntoleratedTaints struct {
	nodeName          string
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
			// PreferNoSchedule is advisory: DaemonSet controller schedules regardless of whether the taint is tolerated
			if taint.Effect == corev1.TaintEffectPreferNoSchedule {
				continue
			}

			if !isTaintTolerated(taint, tolerations) {
				untolerated = append(untolerated, taint)
			}
		}

		if len(untolerated) > 0 {
			result = append(result, nodeWithUntoleratedTaints{
				nodeName:          node.Name,
				untoleratedTaints: untolerated,
			})
		}
	}

	return result
}

func isTaintTolerated(taint corev1.Taint, tolerations []corev1.Toleration) bool {
	for i := range tolerations {
		// true = enableComparisonOperators, allows Lt/Gt operator matching
		if tolerations[i].ToleratesTaint(klog.Background(), &taint, true) {
			return true
		}
	}

	return false
}

func formatToleration(t corev1.Toleration) string {
	key := t.Key
	if key == "" {
		key = "*"
	}

	if t.Operator == corev1.TolerationOpExists || (t.Operator == "" && t.Value == "") {
		if t.Effect == "" {
			return key
		}

		return fmt.Sprintf("%s:%s", key, t.Effect)
	}

	var op string

	switch t.Operator {
	case corev1.TolerationOpLt:
		op = "<"
	case corev1.TolerationOpGt:
		op = ">"
	default:
		op = "="
	}

	if t.Effect == "" {
		return fmt.Sprintf("%s%s%s", key, op, t.Value)
	}

	return fmt.Sprintf("%s%s%s:%s", key, op, t.Value, t.Effect)
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
