package troubleshoot

import (
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/validation"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pullSecretFieldValue = "top-secret"

	dynakubeCrdExistsCheckName             = "checkDynakubeCrdExists"
	getSelectedDynakubeCheckName           = "getSelectedDynakube"
	apiUrlCheckName                        = "apiUrl"
	apiSecretCheckName                     = "apiSecret"
	dynatraceApiSecretHasApiTokenCheckName = "dynatraceApiSecretHasApiToken"
	pullSecretExistsCheckName              = "pullSecretExists"
	pullSecretHasRequiredTokensCheckName   = "pullSecretHasRequiredTokens"
	proxySecretCheckName                   = "proxySecret"
)

func checkDynakube(results ChecksResults, troubleshootCtx *troubleshootContext) error {
	log = newTroubleshootLogger("[dynakube  ] ")

	logNewCheckf("checking if '%s:%s' Dynakube is configured correctly", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)

	err := runChecks(results, troubleshootCtx, getDynakubeChecks())
	if err != nil {
		return errors.Wrapf(err, "'%s:%s' Dynakube isn't valid. %s",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName, dynakubeNotValidMessage())
	}

	logOkf("'%s:%s' Dynakube is valid", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func getDynakubeChecks() []*Check {
	dynakubeCrdExistsCheck := &Check{
		Name: dynakubeCrdExistsCheckName,
		Do:   checkDynakubeCrdExists,
	}

	selectedDynakubeCheck := &Check{
		Name:          getSelectedDynakubeCheckName,
		Do:            getSelectedDynakube,
		Prerequisites: []*Check{dynakubeCrdExistsCheck},
	}

	apiUrlCheck := &Check{
		Name:          apiUrlCheckName,
		Do:            checkApiUrl,
		Prerequisites: []*Check{dynakubeCrdExistsCheck},
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

	return []*Check{dynakubeCrdExistsCheck, selectedDynakubeCheck, apiUrlCheck, apiSecretCheck, ifDynatraceApiSecretHasApiTokenCheck, pullSecretExistsCheck, pullSecretHasRequiredTokensCheck, proxySecretIfItExistsCheck}
}

func dynakubeNotValidMessage() string {
	return fmt.Sprintf(
		"Target namespace and dynakube can be changed by providing '--%s <namespace>' or '--%s <dynakube>' parameters.",
		namespaceFlagName, dynakubeFlagName)
}

func checkDynakubeCrdExists(troubleshootCtx *troubleshootContext) error {
	dynakubeList := &dynatracev1beta1.DynaKubeList{}
	err := troubleshootCtx.apiReader.List(troubleshootCtx.context, dynakubeList, &client.ListOptions{Namespace: troubleshootCtx.namespaceName})

	if err != nil {
		return determineDynakubeError(err)
	}

	logInfof("CRD for Dynakube exists")
	return nil
}

func determineDynakubeError(err error) error {
	if runtime.IsNotRegisteredError(err) {
		err = errors.Wrap(err, "CRD for Dynakube missing")
	} else {
		err = errors.Wrap(err, "could not list Dynakube")
	}
	return err
}

func getSelectedDynakube(troubleshootCtx *troubleshootContext) error {
	query := kubeobjects.NewDynakubeQuery(troubleshootCtx.apiReader, troubleshootCtx.namespaceName).WithContext(troubleshootCtx.context)
	dynakube, err := query.Get(types.NamespacedName{Namespace: troubleshootCtx.namespaceName, Name: troubleshootCtx.dynakubeName})

	if err != nil {
		return determineSelectedDynakubeError(troubleshootCtx, err)
	}

	troubleshootCtx.dynakube = dynakube

	logInfof("using '%s:%s' Dynakube", troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	return nil
}

func determineSelectedDynakubeError(troubleshootCtx *troubleshootContext, err error) error {
	if k8serrors.IsNotFound(err) {
		err = errors.Wrapf(err,
			"selected '%s:%s' Dynakube does not exist",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
	} else {
		err = errors.Wrapf(err, "could not get Dynakube '%s:%s'",
			troubleshootCtx.namespaceName, troubleshootCtx.dynakubeName)
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
