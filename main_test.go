package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type mockResponseWriter struct {
	buffer   bufio.ReadWriter
	body     *bytes.Buffer // Read and Write methods
	statusCh chan int
	header   *http.Header
}

func (mrw mockResponseWriter) Header() http.Header {
	return *mrw.header
}
func (mrw mockResponseWriter) WriteHeader(status int) {
	mrw.statusCh <- status
}

func (mrw mockResponseWriter) Write(b []byte) (int, error) {
	return mrw.body.Write(b)
}

func (mrw *mockResponseWriter) getBody() []byte {
	return mrw.body.Bytes()
}

//	func (mrw *mockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
//		return , bufio.NewReadWriter(h.in, h.out), nil
//
//
//		h, ok := mrw.buffer.(http.Hijacker)
//		if !ok {
//			return nil, nil, errors.New("hijack not supported")
//		}
//		return h.Hijack()
//	}
func newMockResponseWriter() mockResponseWriter {

	return mockResponseWriter{
		body:     &bytes.Buffer{},
		statusCh: make(chan int, 1),
		header:   &http.Header{},
	}
}

type mockRoundTripper struct {
}

func (mrt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("error")
}
func TestProxyHTTP_GET_OK(t *testing.T) {
	expectedBody := "TEST HTTP GET - 200 OK"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//t.Log("Inside mock server handler...")
		if r.Method == "GET" {
			w.WriteHeader(200)
			_, err := w.Write([]byte(expectedBody))
			if err != nil {
				t.Fatalf("error writing response: %v", err)
			}
		} else {
			w.WriteHeader(400)
			_, _ = w.Write([]byte("This server only accepts GET requests."))
		}
	}))
	defer mockServer.Close()
	targetURL := mockServer.URL
	t.Logf("targetURL: %s", targetURL)
	req := httptest.NewRequest(http.MethodGet, targetURL, nil)
	recorder := httptest.NewRecorder()

	proxyHTTP(recorder, req)
	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", response.StatusCode)
	}
	actualBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(actualBody) != expectedBody {
		t.Fatalf("actualBody: %s != expectedBody: %s", actualBody, expectedBody)
	}
}

func TestProxyTunnel(t *testing.T) {
	expectedBody := "TEST TUNNEL GET - 200 OK"
	mockServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("Inside mock TLS server handler...")
		if r.Method == "GET" {
			w.WriteHeader(200)
			_, err := w.Write([]byte(expectedBody))
			if err != nil {
				t.Fatalf("error writing response: %v", err)
			}
		} else {
			w.WriteHeader(400)
			t.Logf("This server only requests GET requsts")
			_, _ = w.Write([]byte("This server only accepts GET requests."))
		}
	}))
	defer mockServer.Close()
	//targetURL := fmt.Sprintf("https://%s", mockServer.URL)
	targetURL, _ := url.Parse(mockServer.URL)
	t.Logf("targetURL: %s", targetURL)
	req := httptest.NewRequest(http.MethodConnect, targetURL.Path, nil)
	req.Host = targetURL.Host
	req.Header.Set("Host", targetURL.Host)

	//req.Header.Set("Connection", "Keep-Alive")
	recorder := httptest.NewRecorder()
	proxyTunnel(recorder, req)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", response.StatusCode)
	}
	actualBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(actualBody) != expectedBody {
		t.Fatalf("actualBody: %s != expectedBody: %s", actualBody, expectedBody)
	}
}
