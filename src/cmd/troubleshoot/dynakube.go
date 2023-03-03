package troubleshoot

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldValue = "top-secret"

	getSelectedDynakubeCheckName           = "getSelectedDynakube"
	apiUrlSyntaxCheckName                  = "apiUrlSyntax"
	dynatraceApiTokenScopesCheckName       = "dynatraceApiTokenScopes"
	apiUrlLatestAgentVersionCheckName      = "apiUrlLatestAgentVersion"
	dynatraceApiSecretHasApiTokenCheckName = "dynatraceApiSecretHasApiToken"
	pullSecretExistsCheckName              = "pullSecretExists"
	pullSecretHasRequiredTokensCheckName   = "pullSecretHasRequiredTokens"
	proxySecretCheckName                   = "proxySecret"
)

func checkDynakube(results ChecksResults, troubleshootCtx *troubleshootContext) error {
	log = newSubTestLogger("dynakube")

	logNewCheckf("checking if '%s:%s' Dynakube is configured correctly", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Name)

	err := runChecks(results, troubleshootCtx, getDynakubeChecks())
	if err != nil {
		return errors.Wrapf(err, "'%s:%s' Dynakube isn't valid. %s",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Name, dynakubeNotValidMessage())
	}

	logOkf("'%s:%s' Dynakube is valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Name)
	return nil
}

func getDynakubeChecks() []*Check {
	selectedDynakubeCheck := &Check{
		Name: getSelectedDynakubeCheckName,
		Do:   getSelectedDynakube,
	}

	ifDynatraceApiSecretHasApiTokenCheck := &Check{
		Name:          dynatraceApiSecretHasApiTokenCheckName,
		Do:            checkIfDynatraceApiSecretHasApiToken,
		Prerequisites: []*Check{selectedDynakubeCheck},
	}

	apiUrlSyntaxCheck := &Check{
		Name:          apiUrlSyntaxCheckName,
		Do:            checkApiUrlSyntax,
		Prerequisites: []*Check{selectedDynakubeCheck},
	}

	apiUrlTokenScopesCheck := &Check{
		Name:          dynatraceApiTokenScopesCheckName,
		Do:            checkDynatraceApiTokenScopes,
		Prerequisites: []*Check{apiUrlSyntaxCheck, ifDynatraceApiSecretHasApiTokenCheck},
	}

	apiUrlLatestAgentVersionCheck := &Check{
		Name:          apiUrlLatestAgentVersionCheckName,
		Do:            checkApiUrlForLatestAgentVersion,
		Prerequisites: []*Check{apiUrlTokenScopesCheck},
	}

	pullSecretExistsCheck := &Check{
		Name:          pullSecretExistsCheckName,
		Do:            checkPullSecretExists,
		Prerequisites: []*Check{apiUrlLatestAgentVersionCheck},
	}

	pullSecretHasRequiredTokensCheck := &Check{
		Name:          pullSecretHasRequiredTokensCheckName,
		Do:            checkPullSecretHasRequiredTokens,
		Prerequisites: []*Check{pullSecretExistsCheck},
	}

	proxySecretIfItExistsCheck := &Check{
		Name:          proxySecretCheckName,
		Do:            applyProxySettings,
		Prerequisites: []*Check{selectedDynakubeCheck},
	}

	return []*Check{selectedDynakubeCheck, ifDynatraceApiSecretHasApiTokenCheck, apiUrlSyntaxCheck, apiUrlTokenScopesCheck, apiUrlLatestAgentVersionCheck, pullSecretExistsCheck, pullSecretHasRequiredTokensCheck, proxySecretIfItExistsCheck}
}

func dynakubeNotValidMessage() string {
	return fmt.Sprintf(
		"Target namespace and dynakube can be changed by providing '--%s <namespace>' or '--%s <dynakube>' parameters.",
		namespaceFlagName, dynakubeFlagName)
}

func getSelectedDynakube(troubleshootCtx *troubleshootContext) error {
	var dynakube dynatracev1beta1.DynaKube
	err := troubleshootCtx.apiReader.Get(
		troubleshootCtx.context,
		client.ObjectKey{
			Name:      troubleshootCtx.dynakube.Name,
			Namespace: troubleshootCtx.namespaceName,
		},
		&dynakube,
	)

	if err != nil {
		return determineSelectedDynakubeError(troubleshootCtx, err)
	}

	troubleshootCtx.dynakube = dynakube

	logInfof("using '%s:%s' Dynakube", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Name)
	return nil
}

func determineSelectedDynakubeError(troubleshootCtx *troubleshootContext, err error) error {
	if k8serrors.IsNotFound(err) {
		err = errors.Wrapf(err,
			"selected '%s:%s' Dynakube does not exist",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Name)
	} else {
		err = errors.Wrapf(err, "could not get Dynakube '%s:%s'",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Name)
	}
	return err
}

func checkIfDynatraceApiSecretHasApiToken(troubleshootCtx *troubleshootContext) error {
	tokenReader := token.NewReader(troubleshootCtx.apiReader, &troubleshootCtx.dynakube)
	tokens, err := tokenReader.ReadTokens(troubleshootCtx.context)
	if err != nil {
		return errors.Wrapf(err, "'%s:%s' secret is missing or invalid", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	}

	_, hasApiToken := tokens[dtclient.DynatraceApiToken]
	if !hasApiToken {
		return errors.New(fmt.Sprintf("'%s' token is missing in '%s:%s' secret", dtclient.DynatraceApiToken, troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens()))
	}

	troubleshootCtx.dynatraceApiSecretTokens = tokens

	logInfof("secret token 'apiToken' exists")
	return nil
}

func checkApiUrlSyntax(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if syntax of API URL is valid")

	validation.SetLogger(log)
	if validation.NoApiUrl(nil, &troubleshootCtx.dynakube) != "" {
		return errors.New("API URL is invalid")
	}
	if validation.IsInvalidApiUrl(nil, &troubleshootCtx.dynakube) != "" {
		return errors.New("API URL is invalid")
	}

	logInfof("syntax of API URL is valid")
	return nil
}

func checkDynatraceApiTokenScopes(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if token scopes are valid")

	dtc, err := dynatraceclient.NewBuilder(troubleshootCtx.apiReader).
		SetContext(troubleshootCtx.context).
		SetDynakube(troubleshootCtx.dynakube).
		SetTokens(troubleshootCtx.dynatraceApiSecretTokens).
		Build()

	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	tokens := troubleshootCtx.dynatraceApiSecretTokens.SetScopesForDynakube(troubleshootCtx.dynakube)

	if err = tokens.VerifyValues(); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	}

	if err = tokens.VerifyScopes(dtc); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	}

	logInfof("token scopes are valid")
	return nil
}

func checkApiUrlForLatestAgentVersion(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if can pull latest agent version")

	dtc, err := dynatraceclient.NewBuilder(troubleshootCtx.apiReader).
		SetContext(troubleshootCtx.context).
		SetDynakube(troubleshootCtx.dynakube).
		SetTokens(troubleshootCtx.dynatraceApiSecretTokens).
		Build()
	if err != nil {
		return errors.Wrap(err, "failed to build DynatraceAPI client")
	}

	_, err = dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return errors.Wrap(err, "failed to connect to DynatraceAPI")
	}

	logInfof("API token is valid, can pull latest agent version")
	return nil
}

func checkPullSecretExists(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(troubleshootCtx.context, nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakube.PullSecret()})

	if err != nil {
		return errors.Wrapf(err, "'%s:%s' pull secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.PullSecret())
	} else {
		troubleshootCtx.pullSecret = secret
	}

	logInfof("pull secret '%s:%s' exists", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.PullSecret())
	return nil
}

func checkPullSecretHasRequiredTokens(troubleshootCtx *troubleshootContext) error {
	if _, err := kubeobjects.ExtractToken(&troubleshootCtx.pullSecret, dtpullsecret.DockerConfigJson); err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.PullSecret())
	}

	logInfof("secret token '%s' exists", dtpullsecret.DockerConfigJson)
	return nil
}
