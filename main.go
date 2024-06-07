package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

const TIMEOUT_MS = 5000

func handleTunnel(w http.ResponseWriter, r *http.Request) {
	log.Printf("Tunneling connection:\nClient:\t%v\nTarget:\t%v", r.RemoteAddr, r.Host)
	// Establish a connection with the target server
	destConn, err := net.DialTimeout("tcp", r.Host, TIMEOUT_MS*time.Millisecond)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)

	// "hijack" the connection maintained by http to avoid duplicating HTTP headers
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Tunneling (hijacking) not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	// At this point, we have connection to client, and connection to server
	// Forward messages from client->target and target->client
	go transfer(clientConn, destConn)
	go transfer(destConn, clientConn)
}

func transfer(from io.ReadCloser, to io.WriteCloser) {
	defer to.Close()
	defer from.Close()
	// dest, src
	io.Copy(to, from)
}

// func handleHTTP(w http.ResponseWriter, r *http.Request) {
// 	log.Printf("HTTP connection:\nClient:\t%v\nTarget:\t%v\n", r.RemoteAddr, r.Host)
// 	reqBytes, err := httputil.DumpRequest(r, true)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	log.Printf("Request Body:\n%v\n", string(reqBytes))
// 	res, err := http.DefaultTransport.RoundTrip(r)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusServiceUnavailable)
// 		return
// 	}

// 	middleResBytes, err := httputil.DumpResponse(res, true)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	log.Printf("Response:\n%v\n", string(middleResBytes))
// 	defer res.Body.Close()
// 	finalRes := filterResponse(*res)
// 	finalResBytes, err := httputil.DumpResponse(&finalRes, true)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	log.Printf("Final Response:\n%v\n", string(finalResBytes))
// 	copyHeader(finalRes.Header, w.Header())
// 	w.WriteHeader(finalRes.StatusCode)
// 	io.Copy(w, finalRes.Body)
// }

func getHTTPHandler(conf config) func(http.ResponseWriter, *http.Request) {
	handler :=
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP connection:\nClient:\t%v\nTarget:\t%v\n", r.RemoteAddr, r.Host)
			reqBytes, err := httputil.DumpRequest(r, conf.logRequestBody)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("Request Body:\n%v\n", string(reqBytes))
			res, err := http.DefaultTransport.RoundTrip(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}

			middleResBytes, err := httputil.DumpResponse(res, conf.logResponseBody)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Printf("Response:\n%v\n", string(middleResBytes))
			defer res.Body.Close()
			finalRes := filterResponse(*res)
			finalResBytes, err := httputil.DumpResponse(&finalRes, conf.logResponseBody)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("Final Response:\n%v\n", string(finalResBytes))
			copyHeader(finalRes.Header, w.Header())
			w.WriteHeader(finalRes.StatusCode)
			io.Copy(w, finalRes.Body)
		}
	return handler
}

// type set[T comparable] map[T]struct{}

// func (s set[T]) has(val T) bool {
// 	_, ok := s[val]
// 	return ok
// }
// func (s set[T]) insert(val T) bool {
// 	if s.has(val) {
// 		return false
// 	}
// 	s[val] = struct{}{}
// 	return true
// }

type set struct {
	entries map[string]struct{}
}

func newSet() *set {
	entries := make(map[string]struct{})
	return &set{entries: entries}
}
func (s set) has(val string) bool {
	_, ok := s.entries[val]
	return ok
}
func (s set) insert(val string) bool {
	if s.has(val) {
		return false
	}
	s.entries[val] = struct{}{}
	return true
}

type headerEntry struct {
	key    string
	values []string
}

func withoutHeaders(in <-chan headerEntry, unwanted set) <-chan headerEntry {
	out := make(chan headerEntry)
	go func() {
		defer close(out)
		for entry := range in {
			if !unwanted.has(entry.key) {
				out <- entry
			}
		}
	}()
	// for header, _ := range unwanted {
	// 	delete(h, header)
	// }
	return out
}
func filterResponse(response http.Response) http.Response {
	filteredRes := response
	headerCh := make(chan headerEntry)
	unwantedHeaders := newSet()
	if unwantedHeaders == nil {
		log.Fatalf("Unwanted Headers set is nil")
	}
	unwantedHeaders.insert("Cookie")
	go func() {
		defer close(headerCh)
		for k, vals := range response.Header {
			headerCh <- headerEntry{key: http.CanonicalHeaderKey(k), values: vals}
		}
	}()
	filteredHeaderCh := withoutHeaders(headerCh, *unwantedHeaders)
	go func() {
		for h := range filteredHeaderCh {
			for _, v := range h.values {
				filteredRes.Header.Add(h.key, v)
			}
		}
	}()
	return filteredRes
}

func copyHeader(from, to http.Header) {
	for k, headers := range from {
		for _, header := range headers {
			to.Add(k, header)
		}
	}
}

type config struct {
	logRequestBody  bool
	logResponseBody bool
}

func (c config) String() string {
	return fmt.Sprintf("Log Req Body:\t%v\nLog Res Body:\t%v\n",
		c.logRequestBody,
		c.logResponseBody)
}
func run(conf config) {
	fmt.Println("GO FORWARD HTTP(S) PROXY")
	fmt.Printf("using config:\n%v\n", conf)
	handleHTTP := getHTTPHandler(conf)

	server := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunnel(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
		// Disables HTTP/2
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	log.Fatal(server.ListenAndServe())
}

func main() {
	conf := config{
		logRequestBody:  false,
		logResponseBody: false,
	}
	run(conf)
}
