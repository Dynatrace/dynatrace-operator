package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
)

const stacktraceKey = "stacktrace"
const errorVerboseKey = "errorVerbose"

type errorPrettify struct{}

func (pretty *errorPrettify) Write(payload []byte) (int, error) {
	return pretty.writeToWriter(payload, os.Stderr)
}

func (pretty *errorPrettify) writeToWriter(payload []byte, writer io.Writer) (int, error) {
	message := string(payload)
	payload, err := replaceDuplicatedStacktrace(payload)
	if err != nil {
		return fmt.Fprint(writer, message)
	}
	return fmt.Fprint(writer, string(payload))
}

func replaceDuplicatedStacktrace(payload []byte) ([]byte, error) {
	var document map[string]interface{}
	err := json.Unmarshal(payload, &document)
	if err != nil {
		// If message is not json, just Write without modification
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
