package connectioninfo

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	TenantUUIDKey             = "tenant-uuid"
	TenantTokenKey            = "tenant-token"
	CommunicationEndpointsKey = "communication-endpoints"

	TokenBasePath         = "/var/lib/dynatrace/secrets/tokens"
	TenantTokenMountPoint = TokenBasePath + "/tenant-token"

	TenantSecretVolumeName = "connection-info-secret"

	EnvDTServer = "DT_SERVER"
	EnvDTTenant = "DT_TENANT"
)

func IsTenantSecretPresent(ctx context.Context, secrets k8ssecret.QueryObject, secretNamespacedName types.NamespacedName) (bool, error) {
	log := logd.FromContext(ctx)

	_, err := secrets.Get(ctx, secretNamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating secret, because missing", "secretName", secretNamespacedName.Name)

			return false, nil
		}

		return false, err
	}

	return true, nil
}

func BuildTenantSecret(dk *dynakube.DynaKube, componentName string, secretName string, tenantToken string) (*corev1.Secret, error) {
	secretData := ExtractSensitiveData(tenantToken)

	coreLabels := k8slabel.NewCoreLabels(dk.Name, componentName)

	return k8ssecret.Build(dk, secretName, secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
}

func ExtractSensitiveData(tenantToken string) map[string][]byte {
	data := map[string][]byte{
		TenantTokenKey: []byte(tenantToken),
	}

	return data
}
