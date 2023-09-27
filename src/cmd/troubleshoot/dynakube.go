package troubleshoot

import (
	"context"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	dynakubevalidation "github.com/Dynatrace/dynatrace-operator/src/webhook/validation/dynakube"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldValue = "top-secret"
)

const dynakubeCheckLoggerName = "dynakube"

func checkDynakube(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string, dynakube *dynatracev1beta1.DynaKube) error {
	dynatraceApiSecretTokens, err := checkIfDynatraceApiSecretHasApiToken(ctx, baseLog, apiReader, namespaceName, dynakube)
	if err != nil {
		return err
	}
	err = checkApiUrlSyntax(baseLog, dynakube)
	if err != nil {
		return err
	}
	err = checkDynatraceApiTokenScopes(ctx, baseLog, apiReader, namespaceName, dynatraceApiSecretTokens, dynakube)
	if err != nil {
		return err
	}

	err = checkApiUrlForLatestAgentVersion(ctx, baseLog, apiReader, dynakube, dynatraceApiSecretTokens)
	if err != nil {
		return err
	}
	pullSecret, err := checkPullSecretExists(ctx, baseLog, apiReader, namespaceName, dynakube)
	if err != nil {
		return err
	}
	err = checkPullSecretHasRequiredTokens(baseLog, namespaceName, dynakube, pullSecret)
	if err != nil {
		return err
	}
	return nil
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

func checkIfDynatraceApiSecretHasApiToken(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string, dynakube *dynatracev1beta1.DynaKube) (token.Tokens, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	tokenReader := token.NewReader(apiReader, dynakube)
	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "'%s:%s' secret is missing or invalid", namespaceName, dynakube.Tokens())
	}

	_, hasApiToken := tokens[dtclient.DynatraceApiToken]
	if !hasApiToken {
		return nil, errors.New(fmt.Sprintf("'%s' token is missing in '%s:%s' secret", dtclient.DynatraceApiToken, namespaceName, dynakube.Tokens()))
	}

	logInfof(log, "secret token 'apiToken' exists")
	return tokens, nil
}

func checkApiUrlSyntax(baseLog logr.Logger, dynakube *dynatracev1beta1.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if syntax of API URL is valid")

	dynakubevalidation.SetLogger(log)
	if dynakubevalidation.NoApiUrl(nil, dynakube) != "" {
		return errors.New("API URL is invalid")
	}
	if dynakubevalidation.IsInvalidApiUrl(nil, dynakube) != "" {
		return errors.New("API URL is invalid")
	}

	logInfof(log, "syntax of API URL is valid")
	return nil
}

func checkDynatraceApiTokenScopes(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string, dynatraceApiSecretTokens token.Tokens, dynakube *dynatracev1beta1.DynaKube) error {
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
		return errors.Wrapf(err, "invalid '%s:%s' secret", namespaceName, dynakube.Tokens())
	}

	if err = tokens.VerifyScopes(dtc); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", namespaceName, dynakube.Tokens())
	}

	logInfof(log, "token scopes are valid")
	return nil
}

func checkApiUrlForLatestAgentVersion(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, dynatraceApiSecretTokens token.Tokens) error {
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

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errors.Wrap(err, "failed to connect to DynatraceAPI")
	}

	logInfof(log, "API token is valid, can pull latest agent version")
	return nil
}

func checkPullSecretExists(ctx context.Context, baseLog logr.Logger, apiReader client.Reader, namespaceName string, dynakube *dynatracev1beta1.DynaKube) (v1.Secret, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	query := kubeobjects.NewSecretQuery(ctx, nil, apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: namespaceName, Name: dynakube.PullSecretName()})

	if err != nil {
		return v1.Secret{}, errors.Wrapf(err, "'%s:%s' pull secret is missing", namespaceName, dynakube.PullSecretName())
	}
	logInfof(log, "pull secret '%s:%s' exists", namespaceName, dynakube.PullSecretName())
	return secret, nil
}

func checkPullSecretHasRequiredTokens(baseLog logr.Logger, namespaceName string, dynakube *dynatracev1beta1.DynaKube, pullSecret v1.Secret) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	if _, err := kubeobjects.ExtractToken(&pullSecret, dtpullsecret.DockerConfigJson); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", namespaceName, dynakube.PullSecretName())
	}

	logInfof(log, "secret token '%s' exists", dtpullsecret.DockerConfigJson)
	return nil
}
