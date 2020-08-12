module github.com/Dynatrace/dynatrace-activegate-operator

go 1.13

require (
	github.com/cosiner/argv v0.1.0 // indirect
	github.com/go-delve/delve v1.4.1 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/google/go-containerregistry v0.1.1
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/operator-framework/operator-sdk v0.17.1-0.20200527074332-363f7b9d2be9
	github.com/peterh/liner v1.2.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.5.1
	go.starlark.net v0.0.0-20200609215844-cd131d1ce9d4 // indirect
	golang.org/x/arch v0.0.0-20200511175325-f7c78586839d // indirect
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubectl v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)
