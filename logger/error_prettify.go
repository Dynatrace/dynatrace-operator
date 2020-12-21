package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
)

const stacktraceKey = "stacktrace"
const errorVerboseKey = "errorVerbose"

type errorPrettify struct{}

func (pretty *errorPrettify) Write(payload []byte) (int, error) {
	message := string(payload)
	payload, err := replaceDuplicatedStacktrace(payload)
	if err != nil {
		return fmt.Fprint(os.Stderr, message)
	}
	return fmt.Fprint(os.Stderr, prettify(payload))
}

func prettify(payload []byte) string {
	message := string(payload)
	message = strings.ReplaceAll(message, "\\n", "\n")
	message = strings.ReplaceAll(message, "\\t", "\t")
	return message
}

func replaceDuplicatedStacktrace(payload []byte) ([]byte, error) {
	var document map[string]interface{}
	err := json.Unmarshal(payload, &document)
	if err != nil {
		// If message is not json, just write without modification
		return nil, errors.WithStack(err)
	}

	document = setErrorVerboseAsStacktrace(document)
	return json.Marshal(document)
}

func setErrorVerboseAsStacktrace(document map[string]interface{}) map[string]interface{} {
	errorVerbose, hasErrorVerbose := document[errorVerboseKey]
	if hasErrorVerbose {
		document[stacktraceKey] = errorVerbose
		delete(document, errorVerboseKey)
	}
	return document
}
