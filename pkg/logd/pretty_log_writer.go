package logd

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func WithWriter(out io.Writer) func(prettifier *prettyLogWriter) {
	return func(prettifier *prettyLogWriter) {
		prettifier.out = out
	}
}

func NewPrettyLogWriter(options ...func(prettifier *prettyLogWriter)) io.Writer {
	pretty := prettyLogWriter{
		out: os.Stdout,
	}
	for _, o := range options {
		o(&pretty)
	}

	return &pretty
}

type prettyLogWriter struct {
	out io.Writer
}

func (pretty *prettyLogWriter) Write(payload []byte) (int, error) {
	if pretty.out == nil {
		return 0, errors.New("no output set on prettyLogWriter")
	}

	prettified, err := removeDuplicatedStacktrace(payload)
	if err != nil {
		return pretty.out.Write(payload)
	}

	if prettified != nil {
		prettified = correctLineEndings(prettified)

		return pretty.out.Write(prettified)
	}

	return pretty.out.Write(payload)
}

func correctLineEndings(payload []byte) []byte {
	message := string(payload)
	message = strings.ReplaceAll(message, "\\n", "\n")
	message = strings.ReplaceAll(message, "\\t", "\t")

	// make sure there is a line break at the end, otherwise logs disappear
	message += "\n"

	return []byte(message)
}

func removeDuplicatedStacktrace(payload []byte) ([]byte, error) {
	var document map[string]any

	err := json.Unmarshal(payload, &document)
	if err != nil {
		// If message is not json, just write without modification
		return payload, errors.WithStack(err)
	}

	document = setErrorVerboseAsStacktrace(document)
	if document != nil {
		return json.Marshal(document)
	}

	return nil, nil
}

func setErrorVerboseAsStacktrace(document map[string]any) map[string]any {
	errorVerbose, hasErrorVerbose := document[errorVerboseKey]
	if hasErrorVerbose {
		document[stacktraceKey] = errorVerbose
		delete(document, errorVerboseKey)
	} else {
		return nil
	}

	return document
}
