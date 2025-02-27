package troubleshoot

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldValue = "top-secret"
)

const dynakubeCheckLoggerName = "dynakube"

func checkDynakube(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) (corev1.Secret, error) {
	dynatraceApiSecretTokens, err := checkIfDynatraceApiSecretHasApiToken(ctx, baseLog, apiReader, dk)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkDynatraceApiTokenScopes(ctx, baseLog, apiReader, dynatraceApiSecretTokens, dk)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkApiUrlForLatestAgentVersion(ctx, baseLog, apiReader, dk, dynatraceApiSecretTokens)
	if err != nil {
		return corev1.Secret{}, err
	}

	pullSecret, err := checkPullSecretExists(ctx, baseLog, apiReader, dk)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkPullSecretHasRequiredTokens(baseLog, dk, pullSecret)
	if err != nil {
		return corev1.Secret{}, err
	}

	return pullSecret, nil
}

func getSelectedDynakube(ctx context.Context, apiReader client.Reader, namespaceName, dynakubeName string) (dynakube.DynaKube, error) {
	var dk dynakube.DynaKube
	err := apiReader.Get(
		ctx,
		client.ObjectKey{
			Name:      dynakubeName,
			Namespace: namespaceName,
		},
		&dk,
	)

	if err != nil {
		return dynakube.DynaKube{}, determineSelectedDynakubeError(namespaceName, dynakubeName, err)
	}

	return dk, nil
}

func dynakubeNotValidMessage() string {
	return fmt.Sprintf(
		"Target namespace and dynakube can be changed by providing '--%s <namespace>' or '--%s <dynakube>' parameters.",
		namespaceFlagName, dynakubeFlagName)
}

func determineSelectedDynakubeError(namespaceName, dynakubeName string, err error) error {
	if k8serrors.IsNotFound(err) {
		err = errors.Wrapf(err,
			"selected '%s:%s' Dynakube does not exist",
			namespaceName, dynakubeName)
	} else {
		err = errors.Wrapf(err, "could not get Dynakube '%s:%s'",
			namespaceName, dynakubeName)
	}

	return err
}

func checkIfDynatraceApiSecretHasApiToken(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) (token.Tokens, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	tokenReader := token.NewReader(apiReader, dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "'%s:%s' secret is missing or invalid", dk.Namespace, dk.Tokens())
	}

	_, hasApiToken := tokens[dtclient.ApiToken]
	if !hasApiToken {
		return nil, errors.New(fmt.Sprintf("'%s' token is missing in '%s:%s' secret", dtclient.ApiToken, dk.Namespace, dk.Tokens()))
	}

	logInfof(log, "secret token 'apiToken' exists")

	return tokens, nil
}

func checkDynatraceApiTokenScopes(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dynatraceApiSecretTokens token.Tokens, dk *dynakube.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if token scopes are valid")

	dtc, err := dynatraceclient.NewBuilder(apiReader).
		SetContext(ctx).
		SetDynakube(*dk).
		SetTokens(dynatraceApiSecretTokens).
		Build()

	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	tokens := dynatraceApiSecretTokens.AddFeatureScopesToTokens()

	if err = tokens.VerifyValues(); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dk.Namespace, dk.Tokens())
	}

	if err = tokens.VerifyScopes(ctx, dtc, *dk); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dk.Namespace, dk.Tokens())
	}

	logInfof(log, "token scopes are valid")

	return nil
}

func checkApiUrlForLatestAgentVersion(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube, dynatraceApiSecretTokens token.Tokens) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if can pull latest agent version")

	dtc, err := dynatraceclient.NewBuilder(apiReader).
		SetContext(ctx).
		SetDynakube(*dk).
		SetTokens(dynatraceApiSecretTokens).
		Build()
	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	_, err = dtc.GetLatestAgentVersion(ctx, dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errors.Wrap(err, "failed to connect to DynatraceAPI")
	}

	logInfof(log, "API token is valid, can pull latest agent version")

	return nil
}

func checkPullSecretExists(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) (corev1.Secret, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	query := secret.Query(nil, apiReader, log)

	pullSecret, err := query.Get(ctx, types.NamespacedName{Namespace: dk.Namespace, Name: dk.PullSecretName()})
	if err != nil {
		return corev1.Secret{}, errors.Wrapf(err, "'%s:%s' pull secret is missing", dk.Namespace, dk.PullSecretName())
	}

	logInfof(log, "pull secret '%s:%s' exists", dk.Namespace, dk.PullSecretName())

	return *pullSecret, nil
}

func checkPullSecretHasRequiredTokens(baseLog logd.Logger, dk *dynakube.DynaKube, pullSecret corev1.Secret) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	if _, err := secret.ExtractToken(&pullSecret, dtpullsecret.DockerConfigJson); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dk.Namespace, dk.PullSecretName())
	}

	logInfof(log, "secret token '%s' exists", dtpullsecret.DockerConfigJson)

	return nil
}
