package main

import (
	"bytes"
	"net/http"
	"testing"
)

type mockResponseWriter struct {
	bytes.Buffer // Read and Write methods
}

func (mrw mockResponseWriter) Header() http.Header {
	return http.Header{}
}
func (mrw mockResponseWriter) WriteHeader(status int) {}

func (mrw mockResponseWriter) getData() []byte {
	return mrw.Bytes()
}

func TestHandleTunnel(t *testing.T) {
	// w := mockResponseWriter{}

	// req, err := http.NewRequest(http.MethodConnect, "")
}
