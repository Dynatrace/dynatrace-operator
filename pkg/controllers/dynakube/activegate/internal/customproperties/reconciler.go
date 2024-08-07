package customproperties

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Suffix     = "custom-properties"
	DataKey    = "customProperties"
	DataPath   = "custom.properties"
	VolumeName = "custom-properties"
	MountPath  = "/var/lib/dynatrace/gateway/config_template/custom.properties"

	ClientInternalSection = "[http.client.internal]"
)

var _ controllers.Reconciler = &Reconciler{}

type Reconciler struct {
	client                    client.Client
	customPropertiesSource    *dynakube.DynaKubeValueSource
	dk                        *dynakube.DynaKube
	customPropertiesOwnerName string
}

func NewReconciler(clt client.Client, dk *dynakube.DynaKube, customPropertiesOwnerName string, customPropertiesSource *dynakube.DynaKubeValueSource) *Reconciler {
	return &Reconciler{
		client:                    clt,
		dk:                        dk,
		customPropertiesSource:    customPropertiesSource,
		customPropertiesOwnerName: customPropertiesOwnerName,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.customPropertiesSource == nil && !r.dk.NeedsCustomNoProxy() {
		return nil
	}

	data, err := r.buildCustomPropertiesValue(ctx)
	if err != nil {
		return err
	}

	customPropertiesSecret, err := secret.Build(r.dk,
		r.buildCustomPropertiesName(r.dk.Name),
		map[string][]byte{
			DataKey: data,
		},
	)
	if err != nil {
		return err
	}

	_, err = secret.Query(r.client, r.client, log).WithOwner(r.dk).CreateOrUpdate(ctx, customPropertiesSecret) // TODO: pass in an apiReader instead of the client 2 times
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) buildCustomPropertiesValue(ctx context.Context) ([]byte, error) {
	value := ""

	if r.customPropertiesSource != nil {
		if r.customPropertiesSource.Value != "" {
			value = r.customPropertiesSource.Value
		} else if r.customPropertiesSource.ValueFrom != "" {
			customPropertiesSecret := &corev1.Secret{}

			err := r.client.Get(ctx, types.NamespacedName{Name: r.customPropertiesSource.ValueFrom, Namespace: r.dk.Namespace}, customPropertiesSecret)
			if err != nil {
				return nil, err
			}

			value = string(customPropertiesSecret.Data[DataKey])
		}
	}

	lines := strings.Split(value, "\n")

	if r.dk.NeedsCustomNoProxy() {
		lines = r.addNonProxyHostsSettingsToValue(lines)
	}

	value = strings.Join(lines, "\n")

	return []byte(value), nil
}

func (r *Reconciler) addNonProxyHostsSettingsToValue(lines []string) []string {
	noProxyValue := r.dk.FeatureNoProxy()
	noProxyValue = strings.ReplaceAll(noProxyValue, ",", "|")
	proxySettings := fmt.Sprintf("%s\nproxy-non-proxy-hosts=%s", ClientInternalSection, noProxyValue)

	found := false

	for i, line := range lines {
		if strings.Contains(line, ClientInternalSection) {
			found = true
			lines[i] = proxySettings

			break
		}
	}

	if !found {
		lines = append(lines, proxySettings)
	}

	return lines
}

func (r *Reconciler) buildCustomPropertiesName(name string) string {
	return fmt.Sprintf("%s-%s-%s", name, r.customPropertiesOwnerName, Suffix)
}
