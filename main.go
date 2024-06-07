package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	fmt.Println("GO FORWARD HTTP(S) PROXY")
	server := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
		// Disables HTTP/2
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	log.Fatal(server.ListenAndServe())
}

const TIMEOUT_MS = 5000

func handleTunneling(w http.ResponseWriter, r *http.Request) {
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

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("HTTP connection:\nClient:\t%v\nTarget:\t%v", r.RemoteAddr, r.Host)
	res, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer res.Body.Close()
	copyHeader(res.Header, w.Header())
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
}

func copyHeader(from, to http.Header) {
	for k, headers := range from {
		for _, header := range headers {
			to.Add(k, header)
		}
	}
}
