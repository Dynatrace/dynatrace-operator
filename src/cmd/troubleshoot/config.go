package troubleshoot

import "github.com/go-logr/logr"

var log logr.Logger

func resetLogger() {
	log = newTroubleshootLogger("")
}
