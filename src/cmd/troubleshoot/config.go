package troubleshoot

import "github.com/go-logr/logr"

var log logr.Logger

func init() {
	resetLog()
}

func resetLog() {
	log = newTroubleshootLogger("[          ]", false)
}
