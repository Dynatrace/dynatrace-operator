package customproperties

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Suffix     = "custom-properties"
	DataKey    = "customProperties"
	DataPath   = "custom.properties"
	VolumeName = "custom-properties"
	MountPath  = "/var/lib/dynatrace/gateway/config_template/custom.properties"

	clientInternalSection = "[http.client.internal]"
	noProxyFieldName      = "proxy-non-proxy-hosts"
)

var _ controllers.Reconciler = &Reconciler{}

type Reconciler struct {
	customPropertiesSource    *value.Source
	dk                        *dynakube.DynaKube
	customPropertiesOwnerName string
	secrets                   k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube, customPropertiesOwnerName string, customPropertiesSource *value.Source) *Reconciler {
	return &Reconciler{
		dk:                        dk,
		customPropertiesSource:    customPropertiesSource,
		customPropertiesOwnerName: customPropertiesOwnerName,
		secrets:                   k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.customPropertiesSource == nil && !r.dk.NeedsCustomNoProxy() {
		if meta.FindStatusCondition(*r.dk.Conditions(), customPropertiesConditionType) == nil {
			return nil
		}

		err := r.secrets.Delete(ctx,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.buildCustomPropertiesName(r.dk.Name),
					Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up custom properties secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), customPropertiesConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	data, err := r.buildCustomPropertiesValue(ctx)
	if err != nil {
		return err
	} else if string(data) == "" {
		return nil
	}

	customPropertiesSecret, err := k8ssecret.Build(r.dk,
		r.buildCustomPropertiesName(r.dk.Name),
		map[string][]byte{
			DataKey: data,
		},
	)
	if err != nil {
		return err
	}

	_, err = r.secrets.WithOwner(r.dk).CreateOrUpdate(ctx, customPropertiesSecret)
	if err != nil {
		return err
	}

	k8sconditions.SetSecretCreated(r.dk.Conditions(), customPropertiesConditionType,
		r.buildCustomPropertiesName(r.dk.Name))

	return nil
}

func (r *Reconciler) buildCustomPropertiesValue(ctx context.Context) ([]byte, error) {
	value := ""

	if r.customPropertiesSource != nil {
		if r.customPropertiesSource.Value != "" {
			value = r.customPropertiesSource.Value
		} else if r.customPropertiesSource.ValueFrom != "" {
			customPropertiesSecret, err := r.secrets.Get(ctx, types.NamespacedName{
				Name:      r.customPropertiesSource.ValueFrom,
				Namespace: r.dk.Namespace})
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
	noProxyValue := r.dk.FF().GetNoProxy()
	noProxyValue = strings.ReplaceAll(noProxyValue, ",", "|")
	proxySettings := fmt.Sprintf("%s\n%s=%s", clientInternalSection, noProxyFieldName, noProxyValue)

	found := false

	for i, line := range lines {
		if strings.Contains(line, clientInternalSection) {
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
