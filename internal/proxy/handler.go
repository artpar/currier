package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProxyHandler handles HTTP and HTTPS proxy requests.
type ProxyHandler struct {
	store     *CaptureStore
	tlsConfig *TLSConfig
	config    Config
}

// NewProxyHandler creates a new proxy handler.
func NewProxyHandler(store *CaptureStore, tlsConfig *TLSConfig, config Config) *ProxyHandler {
	return &ProxyHandler{
		store:     store,
		tlsConfig: tlsConfig,
		config:    config,
	}
}

// ServeHTTP handles incoming proxy requests.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		h.handleHTTPS(w, r)
	} else {
		h.handleHTTP(w, r)
	}
}

// handleHTTP handles plain HTTP requests.
func (h *ProxyHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Check if host should be captured
	if !h.shouldCapture(r.Host) {
		h.proxyRequest(w, r, false, startTime)
		return
	}

	h.proxyRequest(w, r, true, startTime)
}

// proxyRequest forwards the request and optionally captures it.
func (h *ProxyHandler) proxyRequest(w http.ResponseWriter, r *http.Request, capture bool, startTime time.Time) {
	// Build target URL
	targetURL := r.URL
	if !targetURL.IsAbs() {
		targetURL = &url.URL{
			Scheme: "http",
			Host:   r.Host,
			Path:   r.URL.Path,
		}
		if r.URL.RawQuery != "" {
			targetURL.RawQuery = r.URL.RawQuery
		}
	}

	// Read request body
	var reqBody []byte
	if r.Body != nil {
		reqBody, _ = io.ReadAll(io.LimitReader(r.Body, h.config.MaxBodySize))
		r.Body.Close()
	}

	// Create outgoing request
	outReq, err := http.NewRequest(r.Method, targetURL.String(), bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusBadGateway)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			outReq.Header.Add(key, value)
		}
	}

	// Remove hop-by-hop headers
	removeHopByHopHeaders(outReq.Header)

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send request: %v", err), http.StatusBadGateway)

		// Still capture the failed request if enabled
		if capture {
			h.captureError(r, reqBody, err, startTime, false)
		}
		return
	}
	defer resp.Body.Close()

	// Read response body
	var respBody []byte
	if !h.shouldExcludeContentType(resp.Header.Get("Content-Type")) {
		respBody, _ = io.ReadAll(io.LimitReader(resp.Body, h.config.MaxBodySize))
	} else {
		// Read but don't store
		io.Copy(io.Discard, resp.Body)
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	removeHopByHopHeaders(w.Header())

	// Write status and body
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	// Capture if enabled
	if capture {
		h.captureRequest(r, reqBody, resp, respBody, startTime, false)
	}
}

// handleHTTPS handles HTTPS CONNECT requests with MITM.
func (h *ProxyHandler) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	// Check if HTTPS interception is enabled
	if !h.config.EnableHTTPS || h.tlsConfig == nil {
		// Just tunnel without interception
		h.tunnel(w, r)
		return
	}

	// Check if host should be captured
	if !h.shouldCapture(r.Host) {
		h.tunnel(w, r)
		return
	}

	// MITM interception
	h.intercept(w, r)
}

// tunnel creates a TCP tunnel without interception.
func (h *ProxyHandler) tunnel(w http.ResponseWriter, r *http.Request) {
	// Connect to target
	targetConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// Hijack client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to hijack: %v", err), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Send 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Bidirectional copy
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(targetConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, targetConn)
		done <- struct{}{}
	}()
	<-done
}

// intercept performs MITM interception on HTTPS connections.
func (h *ProxyHandler) intercept(w http.ResponseWriter, r *http.Request) {
	// Hijack client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to hijack: %v", err), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Send 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Get host for certificate generation
	host := r.Host
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}
	hostname, _, _ := net.SplitHostPort(host)

	// Get certificate for this host
	cert, err := h.tlsConfig.GetCertForHost(hostname)
	if err != nil {
		return
	}

	// Wrap client connection with TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}
	tlsClientConn := tls.Server(clientConn, tlsConfig)
	if err := tlsClientConn.Handshake(); err != nil {
		return
	}
	defer tlsClientConn.Close()

	// Read requests from client and proxy them
	reader := bufio.NewReader(tlsClientConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return
		}

		// Set the URL scheme and host
		req.URL.Scheme = "https"
		req.URL.Host = hostname
		req.RequestURI = ""

		// Proxy the request
		startTime := time.Now()

		// Read request body
		var reqBody []byte
		if req.Body != nil {
			reqBody, _ = io.ReadAll(io.LimitReader(req.Body, h.config.MaxBodySize))
			req.Body.Close()
		}

		// Create outgoing request
		outReq, err := http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(reqBody))
		if err != nil {
			writeErrorResponse(tlsClientConn, http.StatusBadGateway, err.Error())
			continue
		}

		// Copy headers
		for key, values := range req.Header {
			for _, value := range values {
				outReq.Header.Add(key, value)
			}
		}
		removeHopByHopHeaders(outReq.Header)

		// Send request
		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.Do(outReq)
		if err != nil {
			writeErrorResponse(tlsClientConn, http.StatusBadGateway, err.Error())
			h.captureError(req, reqBody, err, startTime, true)
			continue
		}

		// Read response body
		var respBody []byte
		if !h.shouldExcludeContentType(resp.Header.Get("Content-Type")) {
			respBody, _ = io.ReadAll(io.LimitReader(resp.Body, h.config.MaxBodySize))
		} else {
			io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()

		// Write response to client
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		resp.Write(tlsClientConn)

		// Capture
		h.captureRequest(req, reqBody, resp, respBody, startTime, true)
	}
}

// captureRequest creates a CapturedRequest and stores it.
func (h *ProxyHandler) captureRequest(req *http.Request, reqBody []byte, resp *http.Response, respBody []byte, startTime time.Time, isHTTPS bool) {
	capture := &CapturedRequest{
		ID:              uuid.New().String(),
		Timestamp:       startTime,
		Method:          req.Method,
		URL:             req.URL.String(),
		Host:            req.Host,
		Path:            req.URL.Path,
		RequestHeaders:  req.Header,
		RequestBody:     reqBody,
		RequestSize:     int64(len(reqBody)),
		StatusCode:      resp.StatusCode,
		StatusText:      resp.Status,
		ResponseHeaders: resp.Header,
		ResponseBody:    respBody,
		ResponseSize:    int64(len(respBody)),
		Duration:        time.Since(startTime),
		IsHTTPS:         isHTTPS,
	}

	// Get source IP
	if req.RemoteAddr != "" {
		host, port, err := net.SplitHostPort(req.RemoteAddr)
		if err == nil {
			capture.SourceIP = host
			fmt.Sscanf(port, "%d", &capture.SourcePort)
		}
	}

	h.store.Add(capture)
}

// captureError captures a failed request.
func (h *ProxyHandler) captureError(req *http.Request, reqBody []byte, err error, startTime time.Time, isHTTPS bool) {
	capture := &CapturedRequest{
		ID:             uuid.New().String(),
		Timestamp:      startTime,
		Method:         req.Method,
		URL:            req.URL.String(),
		Host:           req.Host,
		Path:           req.URL.Path,
		RequestHeaders: req.Header,
		RequestBody:    reqBody,
		RequestSize:    int64(len(reqBody)),
		StatusCode:     0,
		Duration:       time.Since(startTime),
		IsHTTPS:        isHTTPS,
		Error:          err.Error(),
	}

	h.store.Add(capture)
}

// shouldCapture checks if the host should be captured based on config.
func (h *ProxyHandler) shouldCapture(host string) bool {
	// Strip port
	hostname, _, _ := net.SplitHostPort(host)
	if hostname == "" {
		hostname = host
	}
	hostname = strings.ToLower(hostname)

	// Check exclude list
	for _, exclude := range h.config.ExcludeHosts {
		if matchHost(hostname, strings.ToLower(exclude)) {
			return false
		}
	}

	// If include list is specified, only capture those hosts
	if len(h.config.IncludeHosts) > 0 {
		for _, include := range h.config.IncludeHosts {
			if matchHost(hostname, strings.ToLower(include)) {
				return true
			}
		}
		return false
	}

	return true
}

// shouldExcludeContentType checks if the content type should be excluded from capture.
func (h *ProxyHandler) shouldExcludeContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	for _, exclude := range h.config.ExcludeContentTypes {
		if strings.HasPrefix(contentType, strings.ToLower(exclude)) {
			return true
		}
	}
	return false
}

// matchHost checks if a hostname matches a pattern (supports * wildcard prefix).
func matchHost(hostname, pattern string) bool {
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(hostname, suffix)
	}
	return hostname == pattern
}

// removeHopByHopHeaders removes hop-by-hop headers that shouldn't be forwarded.
func removeHopByHopHeaders(header http.Header) {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

// writeErrorResponse writes an HTTP error response.
func writeErrorResponse(w io.Writer, statusCode int, message string) {
	fmt.Fprintf(w, "HTTP/1.1 %d %s\r\nContent-Length: %d\r\n\r\n%s",
		statusCode, http.StatusText(statusCode), len(message), message)
}
