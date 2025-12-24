package script

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// LogHandler is a function that handles log output from scripts.
type LogHandler func(message string)

// RequestSender is a function that sends an HTTP request from within a script.
type RequestSender func(options map[string]interface{}) (map[string]interface{}, error)

// Scope provides the execution context for scripts with the currier.* API.
type Scope struct {
	mu     sync.RWMutex
	engine *Engine

	// Request data
	requestMethod  string
	requestURL     string
	requestHeaders map[string]string
	requestBody    string

	// Response data
	responseStatus     int
	responseStatusText string
	responseHeaders    map[string]string
	responseBody       string
	responseTime       int64
	responseSize       int64

	// Variables
	variables      map[string]string
	localVariables map[string]string

	// Environment
	environmentName      string
	environmentVariables map[string]string

	// Handlers
	logHandler    LogHandler
	requestSender RequestSender
}

// NewScope creates a new script execution scope.
func NewScope() *Scope {
	s := &Scope{
		engine:               NewEngine(),
		requestHeaders:       make(map[string]string),
		responseHeaders:      make(map[string]string),
		variables:            make(map[string]string),
		localVariables:       make(map[string]string),
		environmentVariables: make(map[string]string),
	}
	s.setupCurrierAPI()
	return s
}

// Engine returns the underlying JavaScript engine.
func (s *Scope) Engine() *Engine {
	return s.engine
}

// setupCurrierAPI sets up the currier.* global object.
func (s *Scope) setupCurrierAPI() {
	s.engine.RegisterObject("currier", s.buildCurrierObject())
}

// buildCurrierObject builds the currier.* API object.
// Must be called without holding the lock.
func (s *Scope) buildCurrierObject() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.buildCurrierObjectLocked()
}

// buildCurrierObjectLocked builds the currier object. Caller must hold at least RLock.
func (s *Scope) buildCurrierObjectLocked() map[string]interface{} {
	currier := make(map[string]interface{})

	// Request object
	currier["request"] = s.createRequestObjectLocked()

	// Response object
	currier["response"] = s.createResponseObjectLocked()

	// Variables proxy (copy current values)
	currier["variables"] = s.createVariablesProxyLocked()
	currier["getVariable"] = s.getVariableFunc()
	currier["setVariable"] = s.setVariableFunc()
	currier["setLocalVariable"] = s.setLocalVariableFunc()

	// Environment
	currier["environment"] = s.createEnvironmentObjectLocked()

	// Logging
	currier["log"] = s.logFunc()

	// Utilities
	currier["base64"] = s.createBase64Object()
	currier["crypto"] = s.createCryptoObject()

	// Request sending
	currier["sendRequest"] = s.sendRequestFunc()

	return currier
}

// createRequestObjectLocked creates the currier.request object. Caller must hold at least RLock.
func (s *Scope) createRequestObjectLocked() map[string]interface{} {
	// Copy current values
	method := s.requestMethod
	url := s.requestURL
	body := s.requestBody
	headers := make(map[string]string)
	for k, v := range s.requestHeaders {
		headers[k] = v
	}

	return map[string]interface{}{
		"method":  method,
		"url":     url,
		"headers": headers,
		"body":    body,

		"setHeader": func(key, value string) {
			s.mu.Lock()
			s.requestHeaders[key] = value
			s.mu.Unlock()
			// Note: Changes are visible to Go code immediately but not within the same JS execution
		},

		"setBody": func(newBody string) {
			s.mu.Lock()
			s.requestBody = newBody
			s.mu.Unlock()
		},

		"setUrl": func(newUrl string) {
			s.mu.Lock()
			s.requestURL = newUrl
			s.mu.Unlock()
		},
	}
}

// createResponseObjectLocked creates the currier.response object. Caller must hold at least RLock.
func (s *Scope) createResponseObjectLocked() map[string]interface{} {
	// Copy current values
	status := s.responseStatus
	statusText := s.responseStatusText
	body := s.responseBody
	time := s.responseTime
	size := s.responseSize
	headers := make(map[string]string)
	for k, v := range s.responseHeaders {
		headers[k] = v
	}

	return map[string]interface{}{
		"status":     status,
		"statusText": statusText,
		"headers":    headers,
		"body":       body,
		"time":       time,
		"size":       size,

		"json": func() interface{} {
			var result interface{}
			if err := json.Unmarshal([]byte(body), &result); err != nil {
				return nil
			}
			return result
		},
	}
}

// createVariablesProxyLocked creates a copy of variables. Caller must hold at least RLock.
func (s *Scope) createVariablesProxyLocked() map[string]string {
	result := make(map[string]string)
	for k, v := range s.localVariables {
		result[k] = v
	}
	for k, v := range s.variables {
		result[k] = v
	}
	return result
}

// createEnvironmentObjectLocked creates the currier.environment object. Caller must hold at least RLock.
func (s *Scope) createEnvironmentObjectLocked() map[string]interface{} {
	name := s.environmentName

	return map[string]interface{}{
		"name": name,

		"get": func(key string) string {
			s.mu.RLock()
			defer s.mu.RUnlock()
			return s.environmentVariables[key]
		},

		"set": func(key, value string) {
			s.mu.Lock()
			s.environmentVariables[key] = value
			s.mu.Unlock()
		},
	}
}

func (s *Scope) getVariableFunc() func(string) string {
	return func(key string) string {
		s.mu.RLock()
		defer s.mu.RUnlock()

		if v, ok := s.variables[key]; ok {
			return v
		}
		if v, ok := s.localVariables[key]; ok {
			return v
		}
		return ""
	}
}

func (s *Scope) setVariableFunc() func(string, string) {
	return func(key, value string) {
		s.mu.Lock()
		s.variables[key] = value
		s.mu.Unlock()
	}
}

func (s *Scope) setLocalVariableFunc() func(string, string) {
	return func(key, value string) {
		s.mu.Lock()
		s.localVariables[key] = value
		s.mu.Unlock()
	}
}

func (s *Scope) logFunc() func(args ...interface{}) {
	return func(args ...interface{}) {
		parts := make([]string, len(args))
		for i, arg := range args {
			parts[i] = fmt.Sprintf("%v", arg)
		}
		message := strings.Join(parts, " ")

		s.mu.RLock()
		handler := s.logHandler
		s.mu.RUnlock()

		if handler != nil {
			handler(message)
		}
	}
}

// createBase64Object creates the currier.base64 object.
func (s *Scope) createBase64Object() map[string]interface{} {
	return map[string]interface{}{
		"encode": func(input string) string {
			return base64.StdEncoding.EncodeToString([]byte(input))
		},
		"decode": func(input string) string {
			decoded, err := base64.StdEncoding.DecodeString(input)
			if err != nil {
				return ""
			}
			return string(decoded)
		},
	}
}

// createCryptoObject creates the currier.crypto object.
func (s *Scope) createCryptoObject() map[string]interface{} {
	return map[string]interface{}{
		"md5": func(input string) string {
			hash := md5.Sum([]byte(input))
			return hex.EncodeToString(hash[:])
		},
		"sha256": func(input string) string {
			hash := sha256.Sum256([]byte(input))
			return hex.EncodeToString(hash[:])
		},
		"hmac": func(algorithm, key, data string) string {
			mac := hmac.New(sha256.New, []byte(key))
			mac.Write([]byte(data))
			return hex.EncodeToString(mac.Sum(nil))
		},
	}
}

func (s *Scope) sendRequestFunc() func(map[string]interface{}) interface{} {
	return func(options map[string]interface{}) interface{} {
		s.mu.RLock()
		sender := s.requestSender
		s.mu.RUnlock()

		if sender == nil {
			return nil
		}

		result, err := sender(options)
		if err != nil {
			return nil
		}
		return result
	}
}

// refreshCurrierObject updates the currier object in the runtime.
func (s *Scope) refreshCurrierObject() {
	s.engine.RegisterObject("currier", s.buildCurrierObject())
}

// Execute runs a script in this scope.
func (s *Scope) Execute(ctx context.Context, script string) (interface{}, error) {
	// Refresh the currier object to ensure latest state
	s.refreshCurrierObject()
	return s.engine.Execute(ctx, script)
}

// SetRequestMethod sets the request method.
func (s *Scope) SetRequestMethod(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestMethod = method
}

// SetRequestURL sets the request URL.
func (s *Scope) SetRequestURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestURL = url
}

// GetRequestURL gets the request URL.
func (s *Scope) GetRequestURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requestURL
}

// SetRequestHeaders sets the request headers.
func (s *Scope) SetRequestHeaders(headers map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestHeaders = make(map[string]string)
	for k, v := range headers {
		s.requestHeaders[k] = v
	}
}

// GetRequestHeaders gets the request headers.
func (s *Scope) GetRequestHeaders() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range s.requestHeaders {
		result[k] = v
	}
	return result
}

// SetRequestBody sets the request body.
func (s *Scope) SetRequestBody(body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestBody = body
}

// GetRequestBody gets the request body.
func (s *Scope) GetRequestBody() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requestBody
}

// SetResponseStatus sets the response status code.
func (s *Scope) SetResponseStatus(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseStatus = status
}

// SetResponseStatusText sets the response status text.
func (s *Scope) SetResponseStatusText(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseStatusText = text
}

// SetResponseHeaders sets the response headers.
func (s *Scope) SetResponseHeaders(headers map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseHeaders = make(map[string]string)
	for k, v := range headers {
		s.responseHeaders[k] = v
	}
}

// SetResponseBody sets the response body.
func (s *Scope) SetResponseBody(body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseBody = body
}

// SetResponseTime sets the response time in milliseconds.
func (s *Scope) SetResponseTime(time int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseTime = time
}

// SetResponseSize sets the response size in bytes.
func (s *Scope) SetResponseSize(size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseSize = size
}

// SetVariable sets a variable.
func (s *Scope) SetVariable(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.variables[key] = value
}

// GetVariable gets a variable value.
func (s *Scope) GetVariable(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.variables[key]
}

// SetEnvironmentName sets the environment name.
func (s *Scope) SetEnvironmentName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.environmentName = name
}

// SetEnvironmentVariable sets an environment variable.
func (s *Scope) SetEnvironmentVariable(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.environmentVariables[key] = value
}

// GetEnvironmentVariable gets an environment variable.
func (s *Scope) GetEnvironmentVariable(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.environmentVariables[key]
}

// SetLogHandler sets the handler for log output.
func (s *Scope) SetLogHandler(handler LogHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logHandler = handler
}

// SetRequestSender sets the function for sending requests from scripts.
func (s *Scope) SetRequestSender(sender RequestSender) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestSender = sender
}

// Clone creates a copy of the scope.
func (s *Scope) Clone() *Scope {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := &Scope{
		engine:               s.engine.Clone(),
		requestMethod:        s.requestMethod,
		requestURL:           s.requestURL,
		requestHeaders:       make(map[string]string),
		requestBody:          s.requestBody,
		responseStatus:       s.responseStatus,
		responseStatusText:   s.responseStatusText,
		responseHeaders:      make(map[string]string),
		responseBody:         s.responseBody,
		responseTime:         s.responseTime,
		responseSize:         s.responseSize,
		variables:            make(map[string]string),
		localVariables:       make(map[string]string),
		environmentName:      s.environmentName,
		environmentVariables: make(map[string]string),
		logHandler:           s.logHandler,
		requestSender:        s.requestSender,
	}

	for k, v := range s.requestHeaders {
		clone.requestHeaders[k] = v
	}
	for k, v := range s.responseHeaders {
		clone.responseHeaders[k] = v
	}
	for k, v := range s.variables {
		clone.variables[k] = v
	}
	for k, v := range s.localVariables {
		clone.localVariables[k] = v
	}
	for k, v := range s.environmentVariables {
		clone.environmentVariables[k] = v
	}

	clone.setupCurrierAPI()
	return clone
}

// Reset clears all scope data.
func (s *Scope) Reset() {
	s.mu.Lock()
	s.requestMethod = ""
	s.requestURL = ""
	s.requestHeaders = make(map[string]string)
	s.requestBody = ""
	s.responseStatus = 0
	s.responseStatusText = ""
	s.responseHeaders = make(map[string]string)
	s.responseBody = ""
	s.responseTime = 0
	s.responseSize = 0
	s.variables = make(map[string]string)
	s.localVariables = make(map[string]string)
	s.environmentName = ""
	s.environmentVariables = make(map[string]string)
	s.mu.Unlock()

	s.engine.Reset()
	s.setupCurrierAPI()
}
