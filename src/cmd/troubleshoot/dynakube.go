package troubleshoot

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	pullSecretFieldValue = "top-secret"

	getSelectedDynakubeCheckName           = "getSelectedDynakube"
	apiUrlCheckName                        = "apiUrl"
	apiSecretCheckName                     = "apiSecret"
	dynatraceApiSecretHasApiTokenCheckName = "dynatraceApiSecretHasApiToken" //nolint:gosec
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

	apiUrlCheck := &Check{
		Name: apiUrlCheckName,
		Do:   checkApiUrl,
	}

	apiSecretCheck := &Check{
		Name:          apiSecretCheckName,
		Do:            getDynatraceApiSecret,
		Prerequisites: []*Check{selectedDynakubeCheck},
	}

	ifDynatraceApiSecretHasApiTokenCheck := &Check{
		Name:          dynatraceApiSecretHasApiTokenCheckName,
		Do:            checkIfDynatraceApiSecretHasApiToken,
		Prerequisites: []*Check{apiSecretCheck},
	}

	pullSecretExistsCheck := &Check{
		Name:          pullSecretExistsCheckName,
		Do:            checkPullSecretExists,
		Prerequisites: []*Check{selectedDynakubeCheck},
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

	return []*Check{selectedDynakubeCheck, apiUrlCheck, apiSecretCheck, ifDynatraceApiSecretHasApiTokenCheck, pullSecretExistsCheck, pullSecretHasRequiredTokensCheck, proxySecretIfItExistsCheck}
}

func dynakubeNotValidMessage() string {
	return fmt.Sprintf(
		"Target namespace and dynakube can be changed by providing '--%s <namespace>' or '--%s <dynakube>' parameters.",
		namespaceFlagName, dynakubeFlagName)
}

func getSelectedDynakube(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewDynakubeQuery(troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(troubleshootCtx.context)
	dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakube.Name})

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

func checkApiUrl(troubleshootCtx *troubleshootContext) error {
	logInfof("checking if api url is valid")

	validation.SetLogger(log)
	if validation.NoApiUrl(nil, &troubleshootCtx.dynakube) != "" {
		return errors.New("api url is invalid")
	}
	if validation.IsInvalidApiUrl(nil, &troubleshootCtx.dynakube) != "" {
		return errors.New("api url is invalid")
	}

	logInfof("api url is valid")
	return nil
}

func getDynatraceApiSecret(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewSecretQuery(troubleshootCtx.context, nil, troubleshootCtx.apiReader, log)
	secret, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakube.Tokens()})

	if err != nil {
		return errors.Wrapf(err, "'%s:%s' secret is missing", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	} else {
		troubleshootCtx.dynatraceApiSecret = secret
	}

	logInfof("'%s:%s' secret exists", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	return nil
}

func checkIfDynatraceApiSecretHasApiToken(troubleshootCtx *troubleshootContext) error {
	apiToken, err := kubeobjects.ExtractToken(&troubleshootCtx.dynatraceApiSecret, dtclient.DynatraceApiToken)
	if err != nil {
		return errors.Wrapf(err, "invalid '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens())
	}

	if apiToken == "" {
		return errors.New(fmt.Sprintf("'apiToken' token is empty  in '%s:%s' secret", troubleshootCtx.namespaceName, troubleshootCtx.dynakube.Tokens()))
	}

	logInfof("secret token 'apiToken' exists")
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
