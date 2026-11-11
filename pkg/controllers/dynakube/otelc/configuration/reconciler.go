// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
	"path/filepath"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const conditionType string = "OTELCConfigurationConfigMap"

func GetConfigMapName(dkName string) string {
	return dkName + otelcconsts.TelemetryCollectorConfigmapSuffix
}

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, log := logd.NewFromContext(ctx, "config")

	if !dk.TelemetryIngest().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), conditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := k8sconfigmap.Query(r.client, r.apiReader)

		err := query.Delete(ctx, &corev1.ConfigMap{Name: GetConfigMapName(dk.Name), Namespace: dk.Namespace})
		if err != nil {
			log.Error(err, "failed to clean-up OTELC configuration configmap")
		}

		meta.RemoveStatusCondition(dk.Conditions(), conditionType)

		return nil
	}

	return r.reconcileConfigMap(ctx, dk)
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, dk *dynakube.DynaKube) error {
	query := k8sconfigmap.Query(r.client, r.apiReader)

	newConfigMap, err := r.prepareConfigMap(dk)
	if err != nil {
		return err
	}

	changed, err := query.CreateOrUpdate(ctx, newConfigMap)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	} else if changed {
		k8sconditions.SetConfigMapOutdated(dk.Conditions(), conditionType, newConfigMap.Name) // needed so the timestamp updates, will never actually show up in the status
	}

	k8sconditions.SetConfigMapCreatedOrUpdated(dk.Conditions(), conditionType, newConfigMap.Name)

	return nil
}

func (r *Reconciler) prepareConfigMap(dk *dynakube.DynaKube) (*corev1.ConfigMap, error) {
	data, err := r.getData(dk)
	if err != nil {
		return nil, err
	}

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.OTelColComponentLabel).BuildLabels()

	newSecret, err := k8sconfigmap.Build(
		dk,
		GetConfigMapName(dk.Name),
		data,
		k8sconfigmap.SetLabels(coreLabels),
	)
	if err != nil {
		k8sconditions.SetConfigMapGenFailed(dk.Conditions(), conditionType, err)

		return nil, err
	}

	return newSecret, err
}

var (
	transformPodIP = component.MustNewIDWithName("transform", "add-pod-ip")
	batch          = component.MustNewType("batch")
	batchTraces    = component.NewIDWithName(batch, "traces")
	batchMetrics   = component.NewIDWithName(batch, "metrics")
	batchLogs      = component.NewIDWithName(batch, "logs")
)

// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.157.0/processor/batchprocessor/config.go#L16
type batchProcessorConfig struct {
	// Timeout sets the time after which a batch will be sent regardless of size.
	// When this is set to zero, batched data will be sent immediately.
	Timeout time.Duration `mapstructure:"timeout"`

	// SendBatchSize is the size of a batch which after hit, will trigger it to be sent.
	// When this is set to zero, the batch size is ignored and data will be sent immediately
	// subject to only send_batch_max_size.
	SendBatchSize uint32 `mapstructure:"send_batch_size"`

	// SendBatchMaxSize is the maximum size of a batch. It must be larger than SendBatchSize.
	// Larger batches are split into smaller units.
	// Default value is 0, that means no maximum size.
	SendBatchMaxSize uint32 `mapstructure:"send_batch_max_size"`
}

func (r *Reconciler) getData(dk *dynakube.DynaKube) (map[string]string, error) {
	var (
		caFile, tlsCert, tlsKey string
		includeSystemCAs        bool
	)

	if dk.IsAGCertificateNeeded() {
		caFile = otelcconsts.ActiveGateTLSCertVolumePath
	} else if dk.IsCACertificateNeeded() {
		caFile = otelcconsts.TrustedCAVolumePath
		includeSystemCAs = true
	}

	if dk.TelemetryIngest().IsEnabled() && dk.TelemetryIngest().TLSRefName != "" {
		tlsCert = filepath.Join(otelcconsts.CustomTLSCertMountPath, consts.TLSCrtDataName)
		tlsKey = filepath.Join(otelcconsts.CustomTLSCertMountPath, consts.TLSKeyDataName)
	}

	b := otelcgen.NewBuilder()

	for _, protocol := range dk.TelemetryIngest().GetProtocols() {
		b.ReceiverForProtocol(protocol, otelcgen.ReceiverOpts{
			CertFile: tlsCert,
			KeyFile:  tlsKey,
		})
	}

	b.
		MemoryLimiter(otelcgen.MemoryLimiterOpts{LimitPercentage: 70, SpikePercentage: 30}).
		Processor(transformPodIP, buildTransformPodIP()).
		K8sEnrichment(otelcgen.K8sEnrichmentOpts{
			ResourceAttributes: dk.Spec.ResourceAttributes,
			ResourceStatements: []string{
				otelcgen.SetAttrFromEnv("k8s.cluster.uid", "K8S_CLUSTER_UID"),
				otelcgen.SetAttrFromEnv("k8s.cluster.name", "K8S_CLUSTER_NAME"),
				// TODO: decide if these should be removed
				`set(attributes["dt.kubernetes.workload.name"], attributes["k8s.workload.name"])`,
				`set(attributes["dt.kubernetes.workload.kind"], attributes["k8s.workload.kind"])`,
				`set(attributes["dt.entity.kubernetes_cluster"], "${env:DT_ENTITY_KUBERNETES_CLUSTER}")`,
			},
		}).
		Processor(otelcgen.CumulativeToDeltaID, otelcgen.Empty, pipeline.SignalMetrics).
		Processor(batchTraces, &batchProcessorConfig{
			SendBatchSize:    5000,
			SendBatchMaxSize: 5000,
			Timeout:          60 * time.Second,
		}, pipeline.SignalTraces).
		Processor(batchMetrics, &batchProcessorConfig{
			SendBatchSize:    3000,
			SendBatchMaxSize: 3000,
			Timeout:          60 * time.Second,
		}, pipeline.SignalMetrics).
		Processor(batchLogs, &batchProcessorConfig{
			SendBatchSize:    1800,
			SendBatchMaxSize: 2000,
			Timeout:          60 * time.Second,
		}, pipeline.SignalLogs).
		OTLPHTTPExporter("${env:DT_ENDPOINT}", otelcgen.ExporterOpts{
			CAFile:                   caFile,
			APIToken:                 "${env:" + otelcconsts.EnvDataIngestToken + "}",
			IncludeSystemCACertsPool: includeSystemCAs,
		}).
		HealthCheck()

	config := b.Build()

	configBytes, err := config.Marshal()
	if err != nil {
		return nil, err
	}

	configMap := map[string]string{
		otelcconsts.ConfigFieldName: string(configBytes),
	}

	return configMap, nil
}

func buildTransformPodIP() map[string]any {
	return map[string]any{
		"error_mode": "ignore",
		"trace_statements": []map[string]any{
			{
				"context":    "resource",
				"statements": []string{otelcgen.SetAttrIfAbsent("k8s.pod.ip", "ip")},
			},
		},
	}
}
