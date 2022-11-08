package troubleshoot

import "github.com/go-logr/logr"

var log logr.Logger

func resetLog() {
	log = newTroubleshootLogger("", false)
}
