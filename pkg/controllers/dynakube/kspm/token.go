package kspm

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kspm/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ensureKSPMToken(ctx context.Context, client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) error {
	kspmSecretName := dk.Name + "-" + consts.KSPMSecretKey

	query := k8ssecret.Query(client, apiReader, log)
	_, err := query.Get(ctx, types.NamespacedName{Name: kspmSecretName, Namespace: dk.Namespace})

	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("creating new token for kspm", "error", err.Error())

		secretConfig, err := generateKSPMTokenSecret(kspmSecretName, dk)

		if err != nil {
			return err
		}

		err = query.Create(ctx, secretConfig)
		if err != nil {
			log.Info("could not create secret for kspm token", "name", secretConfig.Name)

			return err
		}
	}

	return nil
}

func removeKSPMToken(ctx context.Context, client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) error {
	kspmSecretName := dk.Name + "-" + consts.KSPMSecretKey

	query := k8ssecret.Query(client, apiReader, log)
	secret, err := query.Get(ctx, types.NamespacedName{Name: kspmSecretName, Namespace: dk.Namespace})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		
		return err
	}

	err = query.Delete(ctx, secret)

	if err != nil {
		log.Info("could not delete kspm token", "name", secret.Name)

		return err
	}

	return nil
}

func generateKSPMTokenSecret(name string, dk *dynakube.DynaKube) (secret *v1.Secret, err error) {
	newToken, err := dttoken.New("dt0n01")
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	secretData[consts.KSPMSecretKey] = []byte(newToken.String())
	secretConfig, err := k8ssecret.Build(dk,
		name,
		secretData,
	)

	if err != nil {
		return nil, err
	}

	return secretConfig, nil
}
