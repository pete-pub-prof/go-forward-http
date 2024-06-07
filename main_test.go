package main

import (
	"bytes"
	"io"
	"net/http"

	// "net/url"
	"testing"
)

type mockResponseWriter struct {
	bytes    bytes.Buffer // Read and Write methods
	headerCh chan http.Header
	statusCh chan int
}

func newMockResponseWriter() mockResponseWriter {
	w := mockResponseWriter{
		// Bytes:    make(bytes.Buffer),
		headerCh: make(chan http.Header, 1),
		statusCh: make(chan int, 1),
	}
	return w
}

func (mrw mockResponseWriter) Header() http.Header {
	return http.Header{}
}
func (mrw mockResponseWriter) WriteHeader(status int) {
	mrw.statusCh <- status
}

func (mrw mockResponseWriter) getData() []byte {
	return mrw.bytes.Bytes()
}

type mockTransport struct {
	expectedResponse *http.Response
}

func newMockTransport(expectedResponse *http.Response) http.RoundTripper {
	return &mockTransport{
		expectedResponse: expectedResponse,
	}
}

func (mt *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {

}
func TestProxyHTTP(t *testing.T) {
	// var req *http.Request = new(http.Request)
	// req.Method = http.MethodGet
	// req.URL = new(url.URL)
	// req.URL.Scheme
	var reader io.Reader
	targetURL := "http://fakehost.notreal/some/path"
	req, err := http.NewRequest("GET", targetURL, reader)
	if err != nil {
		t.Logf("Error making dummy request: %v", err)
	}
	// t.Logf("Dummy Req:\nScheme:\t%v\nHost:\t%v", req.URL.Scheme, req.Host)
	t.Logf("Target URL: %v", req.URL)

	// var w mockResponseWriter = newMockResponseWriter()

	// header := <-w.headerCh
	// statusCode := <-w.statusCh
	// req.Response.Request
}

func TestProxyTunnel(t *testing.T) {
	// w := mockResponseWriter{}

	// req, err := http.NewRequest(http.MethodConnect, "")
}
