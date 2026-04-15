package dynatrace

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

// log is kept as a package-level logger because Proxy and Certs are functional option builders
// that receive no context.Context. Threading ctx through them would require a breaking API change.
var (
	log = logd.Get().WithName("dtclient")
)
