package connectioninfo

import (
	"context"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	TenantUUIDKey             = "tenant-uuid"
	TenantTokenKey            = "tenant-token"
	CommunicationEndpointsKey = "communication-endpoints"

	TokenBasePath         = "/var/lib/dynatrace/secrets/tokens"
	TenantTokenMountPoint = TokenBasePath + "/tenant-token"

	TenantSecretVolumeName = "connection-info-secret"

	EnvDtServer = "DT_SERVER"
	EnvDtTenant = "DT_TENANT"
)

func IsTenantSecretPresent(ctx context.Context, secrets k8ssecret.QueryObject, secretNamespacedName types.NamespacedName, log logd.Logger) (bool, error) {
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

func BuildTenantSecret(owner metav1.Object, secretName string, connectionInfo dtclient.ConnectionInfo) (*corev1.Secret, error) {
	secretData := ExtractSensitiveData(connectionInfo)

	return k8ssecret.Build(owner, secretName, secretData)
}

func ExtractSensitiveData(connectionInfo dtclient.ConnectionInfo) map[string][]byte {
	data := map[string][]byte{
		TenantTokenKey: []byte(connectionInfo.TenantToken),
	}

	return data
}
