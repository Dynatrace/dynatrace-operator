package connectioninfo

import (
	"context"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func IsTenantSecretPresent(ctx context.Context, apiReader client.Reader, secretNamespacedName types.NamespacedName, log logger.DtLogger) (bool, error) {
	query := k8ssecret.NewQuery(ctx, nil, apiReader, log)

	_, err := query.Get(secretNamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating secret, because missing", "secretName", secretNamespacedName.Name)

			return false, nil
		}

		return false, err
	}

	return true, nil
}

func BuildTenantSecret(owner metav1.Object, scheme *runtime.Scheme, secretName string, connectionInfo dtclient.ConnectionInfo) (*corev1.Secret, error) {
	secretData := ExtractSensitiveData(connectionInfo)

	return k8ssecret.Create(scheme, owner,
		k8ssecret.NewNameModifier(secretName),
		k8ssecret.NewNamespaceModifier(owner.GetNamespace()),
		k8ssecret.NewDataModifier(secretData))
}

func ExtractSensitiveData(connectionInfo dtclient.ConnectionInfo) map[string][]byte {
	data := map[string][]byte{
		TenantTokenKey: []byte(connectionInfo.TenantToken),
	}

	return data
}
