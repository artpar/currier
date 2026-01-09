package components

import (
	"github.com/artpar/currier/internal/proxy"
)

// SelectCaptureItemMsg is sent when a captured request is selected.
type SelectCaptureItemMsg struct {
	Capture *proxy.CapturedRequest
}

// ProxyStartedMsg is sent when the proxy server starts.
type ProxyStartedMsg struct {
	Address  string
	CertPath string // Path to CA cert for HTTPS interception
}

// ProxyStoppedMsg is sent when the proxy server stops.
type ProxyStoppedMsg struct{}

// ProxyErrorMsg is sent when there's a proxy error.
type ProxyErrorMsg struct {
	Error error
}

// CaptureReceivedMsg is sent when a new capture arrives.
type CaptureReceivedMsg struct {
	Capture *proxy.CapturedRequest
}

// ClearCapturesMsg is sent to clear all captures.
type ClearCapturesMsg struct{}

// ExportCaptureMsg is sent when a capture should be exported to a collection.
type ExportCaptureMsg struct {
	Capture *proxy.CapturedRequest
}

// ToggleProxyMsg is sent to start/stop the proxy.
type ToggleProxyMsg struct{}

// ExportCACertMsg is sent when the CA certificate should be exported.
type ExportCACertMsg struct {
	Path string
}

// RefreshCapturesMsg is sent periodically to refresh the captures list.
type RefreshCapturesMsg struct{}
