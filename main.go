package main

import (
	"context"
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

type userAgent struct {
	ua  string
	pct float32
}
type config struct {
	logRequestBody  bool
	logResponseBody bool
	address         string
	// logTunnelBytes  bool
}

func (c config) String() string {
	return fmt.Sprintf("Log Req Body:\t%v\nLog Res Body:\t%v\nAddress:\t%v",
		c.logRequestBody,
		c.logResponseBody,
		c.address)
}

func defaultConfig() config {
	return config{
		logRequestBody:  false,
		logResponseBody: false,
		address:         ":8888",
	}
}

func proxyTunnel(w http.ResponseWriter, r *http.Request) {
	log.Printf("Tunneling connection:\n\tClient=%v, Target=%v", r.RemoteAddr, r.Host)
	reqBytes, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Tunnel Req Bytes:\n%v\n", reqBytes)
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

func proxyHTTP(w http.ResponseWriter, r *http.Request) {

	conf := defaultConfig()
	log.Printf("HTTP connection:\n\tClient=%v, Target=%v\n", r.RemoteAddr, r.Host)
	reqBytes, err := httputil.DumpRequest(r, conf.logRequestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("HTTP Req:\n%v\n", string(reqBytes))
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

	log.Printf("HTTP Raw Res:\n%v\n", string(middleResBytes))
	defer res.Body.Close()
	finalRes := *res
	finalResBytes, err := httputil.DumpResponse(&finalRes, conf.logResponseBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("HTTP Final Res:\n%v\n", string(finalResBytes))
	copyHeader(finalRes.Header, w.Header())
	w.WriteHeader(finalRes.StatusCode)
	io.Copy(w, finalRes.Body)

}

func copyHeader(from, to http.Header) {
	for k, headers := range from {
		for _, header := range headers {
			to.Add(k, header)
		}
	}
}

func logRequest(r *http.Request, withBody bool) error {
	reqBytes, err := httputil.DumpRequest(r, withBody)
	if err != nil {
		return err
	}
	log.Printf("\n->Request:\n%v\n", string(reqBytes))
	return nil
}
func run() {
	conf := defaultConfig()
	log.Println("GO FORWARD HTTP(S) PROXY")
	log.Printf("using config:\n%v\n", conf)

	server := &http.Server{
		Addr: conf.address,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Connection:\n\tClient=%v, Target=%v", r.RemoteAddr, r.Host)
			logRequest(r, conf.logRequestBody)
			// when proxy=http and target=https, it will tunnel
			if r.Method == http.MethodConnect {
				proxyTunnel(w, r)
			} else {
				// when proxy=http && target=http
				proxyHTTP(w, r)
			}
		}),
		// Disables HTTP/2
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	go func() {
		// log.Fatal(server.ListenAndServeTLS("./certificate.pem", "./privatekey.pem"))
		log.Fatal(server.ListenAndServeTLS("./certificate.pem", "./privatekey.pem"))
	}()
	fmt.Println("Server started, press <Enter> to shutdown")
	fmt.Scanln()
	server.Shutdown(context.Background())
	fmt.Println("Server stopped")
	// log.Fatal(server.ListenAndServe())
}

func main() {
	// logReq := flag.Bool("")
	run()
}
