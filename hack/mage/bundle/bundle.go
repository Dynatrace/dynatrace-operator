package bundle

import (
	"os"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/hack/mage/config"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/manifests"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/prerequisites"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/utils"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Bundle generates bundle manifests and metadata, then validates generated files
func Bundle() error {
	config.OLM = true
	mg.SerialDeps(config.Init, prerequisites.Kustomize, manifests.Kubernetes, manifests.Openshift)

	//sh.Exec(nil, os.Stdout, os.Stdout, "make", "bundle")
	platform := "openshift"
	version := "0.0.1"
	olmImage := "registry.connect.redhat.com/dynatrace/dynatrace-operator:v" + version
	/*
		PLATFORM="${1:-openshift}"
		VERSION="${2:-0.0.1}"
		OLM_IMAGE="${3:-registry.connect.redhat.com/dynatrace/dynatrace-operator:v${VERSION}}"
		BUNDLE_CHANNELS="${4:-}"
		BUNDLE_DEFAULT_CHANNEL="${5:-}"
	*/

	kustomize, err := utils.GetCommand("kustomize")
	if err != nil {
		return err
	}

	operatorSdk, err := utils.GetCommand("operator-sdk")
	if err != nil {
		return err
	}

	sdkParams := []string{
		"--extra-service-accounts dynatrace-dynakube-oneagent",
		"--extra-service-accounts dynatrace-dynakube-oneagent-unprivileged",
		"--extra-service-accounts dynatrace-kubernetes-monitoring",
		"--extra-service-accounts dynatrace-activegate",
	}

	/*
		if [ -n "${BUNDLE_CHANNELS}" ]; then
			SDK_PARAMS+=("${BUNDLE_CHANNELS}")
		fi

		if [ -n "${BUNDLE_DEFAULT_CHANNEL}" ]; then
			SDK_PARAMS+=("${BUNDLE_DEFAULT_CHANNEL}")
		fi
	*/

	err = sh.Run(operatorSdk, "generate", "kustomize", "manifests", "-q", "--apis-dir", "./src/api/")
	if err != nil {
		return err
	}
	err = sh.Run("/bin/bash", "-c", "cd \"config/deploy/"+platform+"\" && "+kustomize+" edit set image quay.io/dynatrace/dynatrace-operator:snapshot=\""+olmImage+"\"")
	if err != nil {
		return err
	}
	err = sh.Run("/bin/bash", "-c", kustomize+" build \"config/olm/"+platform+"\" | "+operatorSdk+" generate bundle --overwrite --version \""+version+"\" "+strings.Join(sdkParams, " "))
	if err != nil {
		return err
	}
	err = sh.Run(operatorSdk, "bundle", "validate", "./bundle")
	if err != nil {
		return err
	}

	err = os.RemoveAll("./config/olm/" + platform + "/" + version)
	if err != nil {
		return err
	}
	err = os.MkdirAll("./config/olm/"+platform+"/"+version, 0o755)
	if err != nil {
		return err
	}

	err = sh.Run("/bin/bash", "-c", "mv ./bundle/* \"./config/olm/"+platform+"/"+version+"\"")
	if err != nil {
		return err
	}
	err = os.Rename("./config/olm/"+platform+"/"+version+"/manifests/dynatrace-operator.clusterserviceversion.yaml", "./config/olm/"+platform+"/"+version+"/manifests/dynatrace-operator.v"+version+".clusterserviceversion.yaml")
	if err != nil {
		return err
	}

	mapper := func(placeholderName string) string {
		switch placeholderName {
		case "VERSION":
			return version
		case "PLATFORM":
			return platform
		}

		return ""
	}

	err = executeCommands([]func() error{
		func() error {
			return os.Rename("./bundle.Dockerfile", os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile", mapper))
		},
		func() error {
			return sh.Run("/bin/bash", "-c", os.Expand("grep -v 'scorecard' \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile\" > \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output\"", mapper))
		},
		func() error {
			return os.Rename(os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output", mapper), os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile", mapper))
		},
		func() error {
			return sh.Run("/bin/bash", "-c", os.Expand("sed \"s/bundle/${VERSION}/\" \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile\" > \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output\"", mapper))
		},
		func() error {
			return os.Rename(os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output", mapper), os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile", mapper))
		},
		func() error {
			return sh.Run("/bin/bash", "-c", os.Expand("awk '/operators.operatorframework.io.metrics.project_layout/ { print; print \"  operators.operatorframework.io.bundle.channel.default.v1: alpha\"; next }1' \"./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml\" > \"./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output\"", mapper))
		},
		func() error {
			return os.Rename(os.Expand("./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output", mapper), os.Expand("./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml", mapper))
		},
		func() error {
			return sh.Run("/bin/bash", "-c", os.Expand("awk '/operators.operatorframework.io.${VERSION}.mediatype.v1/ { print \"LABEL operators.operatorframework.io.bundle.channel.default.v1=alpha\"; print; next }1' \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile\" > \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output\"", mapper))
		},
		func() error {
			return os.Rename(os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output", mapper), os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile", mapper))
		},
		func() error {
			return sh.Run("/bin/bash", "-c", os.Expand("grep -v '# Labels for testing.' \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile\" > \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output\"", mapper))
		},
		func() error {
			return os.Rename(os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output", mapper), os.Expand("./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile", mapper))
		},
	})
	if err != nil {
		return err
	}

	if platform == "openshift" {
		err = executeCommands([]func() error{
			func() error {
				return sh.Run("/bin/bash", "-c", os.Expand("echo 'LABEL com.redhat.openshift.versions=\"v4.8-v4.10\"' >> \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile\"", mapper))
			},
			func() error {
				return sh.Run("/bin/bash", "-c", os.Expand("echo 'LABEL com.redhat.delivery.operator.bundle=true' >> \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile\"", mapper))
			},
			func() error {
				return sh.Run("/bin/bash", "-c", os.Expand("echo 'LABEL com.redhat.delivery.backport=true' >> \"./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile\"", mapper))
			},
			func() error {
				return sh.Run("/bin/bash", "-c", os.Expand("sed 's/\bkubectl\b/oc/g' \"./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml\" > \"./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml.output\"", mapper))
			},
			func() error {
				return os.Rename(os.Expand("./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml.output", mapper), os.Expand("./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml", mapper))
			},
			func() error {
				return sh.Run("/bin/bash", "-c", os.Expand("echo '  com.redhat.openshift.versions: v4.8-v4.10' >> \"./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml\"", mapper))
			},
		})
		if err != nil {
			return err
		}
	}

	return executeCommands([]func() error{
		func() error {
			return sh.Run("/bin/bash", "-c", os.Expand("grep -v 'scorecard' \"./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml\" > \"./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output\"", mapper))
		},
		func() error {
			return sh.Run("/bin/bash", "-c", os.Expand("grep -v '  # Annotations for testing.' \"./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output\" > \"./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml\"", mapper))
		},
		func() error {
			return os.Remove(os.Expand("./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output", mapper))
		},
		func() error {
			return os.Rename(os.Expand("./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml", mapper), os.Expand("./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.clusterserviceversion.yaml", mapper))
		},
	})
}

// BundleMinimal generates bundle manifests and metadata, validates generated files and removes everything but the CSV file
func BundleMinimal() {
	sh.Exec(nil, os.Stdout, os.Stdout, "make", "bundle/minimal")
}

// BundleBuild builds the docker image used for OLM deployment
func BundleBuild() {
	sh.Exec(nil, os.Stdout, os.Stdout, "make", "bundle/build")
}

func executeCommands(commands []func() error) error {
	for _, command := range commands {
		err := command()
		if err != nil {
			return err
		}
	}
	return nil
}
