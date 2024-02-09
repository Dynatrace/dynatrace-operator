package troubleshoot

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
	dynakubevalidation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/validation/dynakube"
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

func checkDynakube(ctx context.Context, baseLog logger.DtLogger, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) (corev1.Secret, error) {
	dynatraceApiSecretTokens, err := checkIfDynatraceApiSecretHasApiToken(ctx, baseLog, apiReader, dynakube)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkApiUrlSyntax(ctx, baseLog, dynakube)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkDynatraceApiTokenScopes(ctx, baseLog, apiReader, dynatraceApiSecretTokens, dynakube)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkApiUrlForLatestAgentVersion(ctx, baseLog, apiReader, dynakube, dynatraceApiSecretTokens)
	if err != nil {
		return corev1.Secret{}, err
	}

	pullSecret, err := checkPullSecretExists(ctx, baseLog, apiReader, dynakube)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkPullSecretHasRequiredTokens(baseLog, dynakube, pullSecret)
	if err != nil {
		return corev1.Secret{}, err
	}

	return pullSecret, nil
}

func getSelectedDynakube(ctx context.Context, apiReader client.Reader, namespaceName, dynakubeName string) (dynatracev1beta1.DynaKube, error) {
	var dynaKube dynatracev1beta1.DynaKube
	err := apiReader.Get(
		ctx,
		client.ObjectKey{
			Name:      dynakubeName,
			Namespace: namespaceName,
		},
		&dynaKube,
	)

	if err != nil {
		return dynatracev1beta1.DynaKube{}, determineSelectedDynakubeError(namespaceName, dynakubeName, err)
	}

	return dynaKube, nil
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

func checkIfDynatraceApiSecretHasApiToken(ctx context.Context, baseLog logger.DtLogger, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) (token.Tokens, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	tokenReader := token.NewReader(apiReader, dynakube)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "'%s:%s' secret is missing or invalid", dynakube.Namespace, dynakube.Tokens())
	}

	_, hasApiToken := tokens[dtclient.ApiToken]
	if !hasApiToken {
		return nil, errors.New(fmt.Sprintf("'%s' token is missing in '%s:%s' secret", dtclient.ApiToken, dynakube.Namespace, dynakube.Tokens()))
	}

	logInfof(log, "secret token 'apiToken' exists")

	return tokens, nil
}

func checkApiUrlSyntax(ctx context.Context, baseLog logger.DtLogger, dynakube *dynatracev1beta1.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if syntax of API URL is valid")

	dynakubevalidation.SetLogger(log)

	if dynakubevalidation.NoApiUrl(ctx, nil, dynakube) != "" {
		return errors.New("API URL is invalid")
	}

	if dynakubevalidation.IsInvalidApiUrl(ctx, nil, dynakube) != "" {
		return errors.New("API URL is invalid")
	}

	logInfof(log, "syntax of API URL is valid")

	return nil
}

func checkDynatraceApiTokenScopes(ctx context.Context, baseLog logger.DtLogger, apiReader client.Reader, dynatraceApiSecretTokens token.Tokens, dynakube *dynatracev1beta1.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if token scopes are valid")

	dtc, err := dynatraceclient.NewBuilder(apiReader).
		SetContext(ctx).
		SetDynakube(*dynakube).
		SetTokens(dynatraceApiSecretTokens).
		Build()

	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	tokens := dynatraceApiSecretTokens.SetScopesForDynakube(*dynakube)

	if err = tokens.VerifyValues(); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dynakube.Namespace, dynakube.Tokens())
	}

	if err = tokens.VerifyScopes(ctx, dtc); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dynakube.Namespace, dynakube.Tokens())
	}

	logInfof(log, "token scopes are valid")

	return nil
}

func checkApiUrlForLatestAgentVersion(ctx context.Context, baseLog logger.DtLogger, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, dynatraceApiSecretTokens token.Tokens) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if can pull latest agent version")

	dtc, err := dynatraceclient.NewBuilder(apiReader).
		SetContext(ctx).
		SetDynakube(*dynakube).
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

func checkPullSecretExists(ctx context.Context, baseLog logger.DtLogger, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube) (corev1.Secret, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	query := secret.NewQuery(ctx, nil, apiReader, log)

	pullSecret, err := query.Get(types.NamespacedName{Namespace: dynakube.Namespace, Name: dynakube.PullSecretName()})
	if err != nil {
		return corev1.Secret{}, errors.Wrapf(err, "'%s:%s' pull secret is missing", dynakube.Namespace, dynakube.PullSecretName())
	}

	logInfof(log, "pull secret '%s:%s' exists", dynakube.Namespace, dynakube.PullSecretName())

	return pullSecret, nil
}

func checkPullSecretHasRequiredTokens(baseLog logger.DtLogger, dynakube *dynatracev1beta1.DynaKube, pullSecret corev1.Secret) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	if _, err := secret.ExtractToken(&pullSecret, dtpullsecret.DockerConfigJson); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dynakube.Namespace, dynakube.PullSecretName())
	}

	logInfof(log, "secret token '%s' exists", dtpullsecret.DockerConfigJson)

	return nil
}
