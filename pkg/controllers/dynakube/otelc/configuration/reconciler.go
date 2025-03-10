package configuration

import (
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	otelcconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/otelc/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8sconfigmap "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/configmap"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.TelemetryIngest().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := k8sconfigmap.Query(r.client, r.apiReader, log)
		err := query.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: GetConfigMapName(r.dk.Name), Namespace: r.dk.Namespace}})

		if err != nil {
			log.Error(err, "failed to clean-up OTELC configuration configmap")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		return nil
	}

	return r.reconcileConfigMap(ctx)
}

func (r *Reconciler) reconcileConfigMap(ctx context.Context) error {
	query := k8sconfigmap.Query(r.client, r.apiReader, log)

	newConfigMap, err := r.prepareConfigMap()
	if err != nil {
		return err
	}

	changed, err := query.CreateOrUpdate(ctx, newConfigMap)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)

		return err
	} else if changed {
		conditions.SetConfigMapOutdated(r.dk.Conditions(), conditionType, newConfigMap.Name) // needed so the timestamp updates, will never actually show up in the status
	}

	conditions.SetConfigMapCreatedOrUpdated(r.dk.Conditions(), conditionType, newConfigMap.Name)

	return nil
}

func (r *Reconciler) prepareConfigMap() (*corev1.ConfigMap, error) {
	data, err := r.getData()
	if err != nil {
		return nil, err
	}

	coreLabels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.OtelCComponentLabel).BuildLabels()

	newSecret, err := k8sconfigmap.Build(r.dk,
		GetConfigMapName(r.dk.Name),
		data,
		k8sconfigmap.SetLabels(coreLabels),
	)
	if err != nil {
		conditions.SetConfigMapGenFailed(r.dk.Conditions(), conditionType, err)

		return nil, err
	}

	return newSecret, err
}

func (r *Reconciler) getData() (map[string]string, error) {
	myPodIp := "${env:MY_POD_IP}"

	options := []otelcgen.Option{
		otelcgen.WithApiToken("${env:" + otelcconsts.EnvDataIngestToken + "}"),
		otelcgen.WithExportersEndpoint("${env:DT_ENDPOINT}"),
	}

	if r.dk.IsAGCertificateNeeded() {
		options = append(options, otelcgen.WithCA(otelcconsts.ActiveGateTLSCertVolumePath))
	} else if r.dk.IsCACertificateNeeded() {
		options = append(options, otelcgen.WithCA(otelcconsts.TrustedCAVolumePath))
	}

	if r.dk.TelemetryIngest().IsEnabled() && r.dk.TelemetryIngest().Spec.TlsRefName != "" {
		options = append(options, otelcgen.WithTLS(filepath.Join(otelcconsts.CustomTlsCertMountPath, consts.TLSCrtDataName), filepath.Join(otelcconsts.CustomTlsCertMountPath, consts.TLSKeyDataName)))
	}

	options = append(options,
		otelcgen.WithExporters(),
		otelcgen.WithProcessors(),
		otelcgen.WithReceivers(),
		otelcgen.WithExtensions(),
		otelcgen.WithServices(),
	)

	config, err := otelcgen.NewConfig(myPodIp, r.dk.TelemetryIngest().GetProtocols(), options...)
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
