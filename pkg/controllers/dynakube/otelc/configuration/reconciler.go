package configuration

import (
	"context"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	if !dk.TelemetryIngest().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), conditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := k8sconfigmap.Query(r.client, r.apiReader, log)

		err := query.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: GetConfigMapName(dk.Name), Namespace: dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up OTELC configuration configmap")
		}

		meta.RemoveStatusCondition(dk.Conditions(), conditionType)

		return nil
	}

	return r.reconcileConfigMap(ctx, dk)
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context, dk *dynakube.DynaKube) error {
	query := k8sconfigmap.Query(r.client, r.apiReader, log)

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

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.OtelCComponentLabel).BuildLabels()

	newSecret, err := k8sconfigmap.Build(dk,
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

func (r *Reconciler) getData(dk *dynakube.DynaKube) (map[string]string, error) {
	myPodIP := "${env:MY_POD_IP}"

	options := []otelcgen.Option{
		otelcgen.WithAPIToken("${env:" + otelcconsts.EnvDataIngestToken + "}"),
		otelcgen.WithExportersEndpoint("${env:DT_ENDPOINT}"),
	}

	if dk.IsAGCertificateNeeded() {
		options = append(options, otelcgen.WithCA(otelcconsts.ActiveGateTLSCertVolumePath))
	} else if dk.IsCACertificateNeeded() {
		options = append(options, otelcgen.WithCA(otelcconsts.TrustedCAVolumePath))
		options = append(options, otelcgen.WithSystemCAs(true))
	}

	if dk.TelemetryIngest().IsEnabled() && dk.TelemetryIngest().TLSRefName != "" {
		options = append(options, otelcgen.WithTLS(filepath.Join(otelcconsts.CustomTLSCertMountPath, consts.TLSCrtDataName), filepath.Join(otelcconsts.CustomTLSCertMountPath, consts.TLSKeyDataName)))
	}

	options = append(options,
		otelcgen.WithExporters(),
		otelcgen.WithProcessors(),
		otelcgen.WithReceivers(),
		otelcgen.WithExtensions(),
		otelcgen.WithServices(),
	)

	config, err := otelcgen.NewConfig(myPodIP, dk.TelemetryIngest().GetProtocols(), options...)
	if err != nil {
		return nil, err
	}

	configBytes, err := config.Marshal()
	if err != nil {
		return nil, err
	}

	configMap := map[string]string{
		otelcconsts.ConfigFieldName: string(configBytes),
	}

	return configMap, nil
}

func GetConfigMapName(dkName string) string {
	return dkName + otelcconsts.TelemetryCollectorConfigmapSuffix
}
