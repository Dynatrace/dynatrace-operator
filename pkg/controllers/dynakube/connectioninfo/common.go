package connectioninfo

import (
	"context"

	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

func ExtractSensitiveData(connectionInfo dtclient.ConnectionInfo) map[string][]byte {
	data := map[string][]byte{
		TenantTokenKey: []byte(connectionInfo.TenantToken),
	}

	return data
}

func SecretNotPresent(ctx context.Context, apiReader client.Reader, secretNamespacedName types.NamespacedName, log logr.Logger) (bool, error) {
	query := k8ssecret.NewQuery(ctx, nil, apiReader, log)

	_, err := query.Get(secretNamespacedName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating secret, because missing", "secretName", secretNamespacedName.Name)

			return true, nil
		}

		return false, err
	}

	return false, nil
}
