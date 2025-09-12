package core

import (
	"reflect"
	"testing"
)

func TestCommonConfig_GET_POST_PUT_DELETE(t *testing.T) {
	config := newTestConfig("http://localhost")
	if reflect.TypeOf(config.GET(t.Context(), "/foo")).String() != "*core.requestBuilder" {
		t.Errorf("GET did not return requestBuilder")
	}
	if reflect.TypeOf(config.POST(t.Context(), "/foo")).String() != "*core.requestBuilder" {
		t.Errorf("POST did not return requestBuilder")
	}
	if reflect.TypeOf(config.PUT(t.Context(), "/foo")).String() != "*core.requestBuilder" {
		t.Errorf("PUT did not return requestBuilder")
	}
	if reflect.TypeOf(config.DELETE(t.Context(), "/foo")).String() != "*core.requestBuilder" {
		t.Errorf("DELETE did not return requestBuilder")
	}
}
