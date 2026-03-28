package customproperties

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
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

type Reconciler struct {
	secrets k8ssecret.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		secrets: k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube, customPropertiesOwnerName string, customPropertiesSource *value.Source) error {
	if customPropertiesSource == nil && !dk.NeedsCustomNoProxy() {
		if meta.FindStatusCondition(*dk.Conditions(), customPropertiesConditionType) == nil {
			return nil
		}

		err := r.secrets.Delete(ctx,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      r.buildCustomPropertiesName(dk.Name, customPropertiesOwnerName),
					Namespace: dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up custom properties secret")
		}

		meta.RemoveStatusCondition(dk.Conditions(), customPropertiesConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	data, err := r.buildCustomPropertiesValue(ctx, dk, customPropertiesSource)
	if err != nil {
		return err
	} else if string(data) == "" {
		return nil
	}

	customPropertiesSecret, err := k8ssecret.Build(dk,
		r.buildCustomPropertiesName(dk.Name, customPropertiesOwnerName),
		map[string][]byte{
			DataKey: data,
		},
	)
	if err != nil {
		return err
	}

	_, err = r.secrets.WithOwner(dk).CreateOrUpdate(ctx, customPropertiesSecret)
	if err != nil {
		return err
	}

	k8sconditions.SetSecretCreated(dk.Conditions(), customPropertiesConditionType,
		r.buildCustomPropertiesName(dk.Name, customPropertiesOwnerName))

	return nil
}

func (r *Reconciler) buildCustomPropertiesValue(ctx context.Context, dk *dynakube.DynaKube, customPropertiesSource *value.Source) ([]byte, error) {
	customPropertiesValue := ""

	if customPropertiesSource != nil {
		if customPropertiesSource.Value != "" {
			customPropertiesValue = customPropertiesSource.Value
		} else if customPropertiesSource.ValueFrom != "" {
			customPropertiesSecret, err := r.secrets.Get(ctx, types.NamespacedName{
				Name:      customPropertiesSource.ValueFrom,
				Namespace: dk.Namespace})
			if err != nil {
				return nil, err
			}

			customPropertiesValue = string(customPropertiesSecret.Data[DataKey])
		}
	}

	lines := strings.Split(customPropertiesValue, "\n")

	if dk.NeedsCustomNoProxy() {
		lines = r.addNonProxyHostsSettingsToValue(dk.FF().GetNoProxy(), lines)
	}

	customPropertiesValue = strings.Join(lines, "\n")

	return []byte(customPropertiesValue), nil
}

func (r *Reconciler) addNonProxyHostsSettingsToValue(ffNoProxy string, lines []string) []string {
	noProxyValue := strings.ReplaceAll(ffNoProxy, ",", "|")
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

func (r *Reconciler) buildCustomPropertiesName(name string, customPropertiesOwnerName string) string {
	return fmt.Sprintf("%s-%s-%s", name, customPropertiesOwnerName, Suffix)
}
