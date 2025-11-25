package troubleshoot

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
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
	dynatraceAPISecretTokens, err := checkIfDynatraceAPISecretHasAPIToken(ctx, baseLog, apiReader, dk)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkDynatraceAPITokenScopes(ctx, baseLog, apiReader, dynatraceAPISecretTokens, dk)
	if err != nil {
		return corev1.Secret{}, err
	}

	err = checkAPIURLForLatestAgentVersion(ctx, baseLog, apiReader, dk, dynatraceAPISecretTokens)
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

func checkIfDynatraceAPISecretHasAPIToken(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) (token.Tokens, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	tokenReader := token.NewReader(apiReader, dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "'%s:%s' secret is missing or invalid", dk.Namespace, dk.Tokens())
	}

	_, hasAPIToken := tokens[dtclient.APIToken]
	if !hasAPIToken {
		return nil, errors.New(fmt.Sprintf("'%s' token is missing in '%s:%s' secret", dtclient.APIToken, dk.Namespace, dk.Tokens()))
	}

	logInfof(log, "secret token 'apiToken' exists")

	return tokens, nil
}

func checkDynatraceAPITokenScopes(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dynatraceAPISecretTokens token.Tokens, dk *dynakube.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if token scopes are valid")

	dtc, err := dynatraceclient.NewBuilder(apiReader).
		SetDynakube(*dk).
		SetTokens(dynatraceAPISecretTokens).
		Build(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	tokens := dynatraceAPISecretTokens.AddFeatureScopesToTokens()

	if err = tokens.VerifyValues(); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dk.Namespace, dk.Tokens())
	}

	var optionalScopes map[string]bool

	if optionalScopes, err = tokens.VerifyScopes(ctx, dtc, *dk); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dk.Namespace, dk.Tokens())
	}

	missingOptionalScopes := []string{}

	for scope, isAvailable := range optionalScopes {
		if !isAvailable {
			missingOptionalScopes = append(missingOptionalScopes, scope)
		}
	}

	if len(missingOptionalScopes) > 0 {
		logInfof(log, "token scopes are valid however some optional scopes are missing so some features may not work: %s", strings.Join(missingOptionalScopes, ", "))
	} else {
		logInfof(log, "token scopes are valid")
	}

	return nil
}

func checkAPIURLForLatestAgentVersion(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube, dynatraceAPISecretTokens token.Tokens) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if can pull latest agent version")

	dtc, err := dynatraceclient.NewBuilder(apiReader).
		SetDynakube(*dk).
		SetTokens(dynatraceAPISecretTokens).
		Build(ctx)
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

	query := k8ssecret.Query(nil, apiReader, log)

	pullSecret, err := query.Get(ctx, types.NamespacedName{Namespace: dk.Namespace, Name: dk.PullSecretName()})
	if err != nil {
		return corev1.Secret{}, errors.Wrapf(err, "'%s:%s' pull secret is missing", dk.Namespace, dk.PullSecretName())
	}

	logInfof(log, "pull secret '%s:%s' exists", dk.Namespace, dk.PullSecretName())

	return *pullSecret, nil
}

func checkPullSecretHasRequiredTokens(baseLog logd.Logger, dk *dynakube.DynaKube, pullSecret corev1.Secret) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	if _, err := k8ssecret.ExtractToken(&pullSecret, dtpullsecret.DockerConfigJSON); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dk.Namespace, dk.PullSecretName())
	}

	logInfof(log, "secret token '%s' exists", dtpullsecret.DockerConfigJSON)

	return nil
}
