package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyHTTP_httptest(t *testing.T) {
	targetURL = "https://"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

}

func TestProxyTunnel(t *testing.T) {

}
