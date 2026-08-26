// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package troubleshoot

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
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

func checkDynakube(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) error {
	dynatraceAPISecretTokens, err := checkIfDynatraceAPISecretHasAPIToken(ctx, baseLog, apiReader, dk)
	if err != nil {
		return err
	}

	err = checkDynatraceAPITokenScopes(ctx, baseLog, apiReader, dynatraceAPISecretTokens, dk)
	if err != nil {
		return err
	}

	err = checkAPIURLForLatestAgentVersion(ctx, baseLog, apiReader, dk, dynatraceAPISecretTokens)
	if err != nil {
		return err
	}

	pullSecrets, err := checkPullSecretsExist(ctx, baseLog, apiReader, dk)
	if err != nil {
		return err
	}

	for _, pullSecret := range pullSecrets {
		if err = checkPullSecretHasRequiredTokens(baseLog, pullSecret); err != nil {
			return err
		}
	}

	return nil
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

	tokens, err := tokenReader.ReadAndVerifyTokens(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "'%s:%s' secret is missing or invalid", dk.Namespace, dk.Tokens())
	}

	_, hasAPIToken := tokens[token.APIKey]
	if !hasAPIToken {
		return nil, errors.New(fmt.Sprintf("'%s' token is missing in '%s:%s' secret", token.APIKey, dk.Namespace, dk.Tokens()))
	}

	logInfof(log, "secret token 'apiToken' exists")

	return tokens, nil
}

func checkDynatraceAPITokenScopes(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dynatraceAPISecretTokens token.Tokens, dk *dynakube.DynaKube) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	logInfof(log, "checking if token scopes are valid")

	dtClient, err := dynatrace.NewClientFromDynakube(ctx, apiReader, dk, dynatraceAPISecretTokens.APIToken().String(), dynatraceAPISecretTokens.PaasToken().String(), "troubleshoot", k8senv.GetOperatorDTClientConnectionTimeout(ctx))
	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	tokens := dynatraceAPISecretTokens.AddFeatureScopesToTokens()

	if err = tokens.VerifyValues(); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", dk.Namespace, dk.Tokens())
	}

	if tokens.HasPlatformToken() {
		logInfof(log, "skipping token scope lookup due to platform token")

		return nil
	}

	var optionalScopes map[string]bool

	if optionalScopes, err = tokens.VerifyScopes(ctx, dtClient.Token, *dk); err != nil {
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

	dtClient, err := dynatrace.NewClientFromDynakube(ctx, apiReader, dk, dynatraceAPISecretTokens.APIToken().String(), dynatraceAPISecretTokens.PaasToken().String(), "troubleshoot", k8senv.GetOperatorDTClientConnectionTimeout(ctx))
	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	_, err = dtClient.Version.GetLatestAgentVersion(ctx, installer.OSUnix, installer.TypeDefault)
	if err != nil {
		return errors.Wrap(err, "failed to connect to DynatraceAPI")
	}

	logInfof(log, "API token is valid, can pull latest agent version")

	return nil
}

func checkPullSecretsExist(ctx context.Context, baseLog logd.Logger, apiReader client.Reader, dk *dynakube.DynaKube) ([]*corev1.Secret, error) {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	query := k8ssecret.Query(nil, apiReader)

	var secrets []*corev1.Secret

	for _, name := range dk.PullSecretNames() {
		pullSecret, err := query.Get(ctx, types.NamespacedName{Namespace: dk.Namespace, Name: name})
		if err != nil {
			return nil, errors.Wrapf(err, "'%s:%s' pull secret is missing", dk.Namespace, name)
		}

		logInfof(log, "pull secret '%s:%s' exists", dk.Namespace, name)

		secrets = append(secrets, pullSecret)
	}

	return secrets, nil
}

func checkPullSecretHasRequiredTokens(baseLog logd.Logger, pullSecret *corev1.Secret) error {
	log := baseLog.WithName(dynakubeCheckLoggerName)

	if _, err := k8ssecret.ExtractToken(pullSecret, dtpullsecret.DockerConfigJSON); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", pullSecret.Namespace, pullSecret.Name)
	}

	logInfof(log, "secret token '%s' exists", dtpullsecret.DockerConfigJSON)

	return nil
}
