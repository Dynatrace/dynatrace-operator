package utils

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCloseBodyAfterRequest(t *testing.T) {
	type args struct {
		response *http.Response
	}

	tests := []struct {
		name string
		args args
	}{
		{`nil http.Response`, args{response: nil}},
		{`nil http.Response`, args{response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CloseBodyAfterRequest(tt.args.response)
		})
	}
}
