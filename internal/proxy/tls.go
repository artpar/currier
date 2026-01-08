package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TLSConfig manages CA certificate and dynamic host certificate generation.
type TLSConfig struct {
	caCert    *x509.Certificate
	caKey     *rsa.PrivateKey
	certCache sync.Map // map[string]*tls.Certificate
}

// NewTLSConfig loads or generates CA certificate for MITM.
func NewTLSConfig(certPath, keyPath string, autoGenerate bool) (*TLSConfig, error) {
	tc := &TLSConfig{}

	// Try to load existing CA
	if err := tc.loadCA(certPath, keyPath); err != nil {
		if !autoGenerate {
			return nil, fmt.Errorf("failed to load CA certificate: %w", err)
		}

		// Generate new CA
		if err := tc.generateCA(certPath, keyPath); err != nil {
			return nil, fmt.Errorf("failed to generate CA certificate: %w", err)
		}
	}

	return tc, nil
}

// loadCA loads CA certificate and key from files.
func (tc *TLSConfig) loadCA(certPath, keyPath string) error {
	// Read certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}

	// Read key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	// Parse certificate
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Parse key
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8
		keyAny, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse CA key: %w", err)
		}
		var ok bool
		key, ok = keyAny.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("CA key is not RSA")
		}
	}

	tc.caCert = cert
	tc.caKey = key
	return nil
}

// generateCA generates a new CA certificate and key.
func (tc *TLSConfig) generateCA(certPath, keyPath string) error {
	// Generate RSA key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Currier Proxy CA",
			Organization: []string{"Currier"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Parse the generated certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse generated CA certificate: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create CA directory: %w", err)
	}

	// Save certificate
	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Save key
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create CA key file: %w", err)
	}
	defer keyFile.Close()

	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return fmt.Errorf("failed to write CA key: %w", err)
	}

	tc.caCert = cert
	tc.caKey = key
	return nil
}

// GetCertForHost returns a TLS certificate for the given host.
// Certificates are cached for performance.
func (tc *TLSConfig) GetCertForHost(host string) (*tls.Certificate, error) {
	// Check cache
	if cached, ok := tc.certCache.Load(host); ok {
		return cached.(*tls.Certificate), nil
	}

	// Generate new certificate
	cert, err := tc.generateHostCert(host)
	if err != nil {
		return nil, err
	}

	// Cache it
	tc.certCache.Store(host, cert)
	return cert, nil
}

// generateHostCert generates a certificate for the given host, signed by the CA.
func (tc *TLSConfig) generateHostCert(host string) (*tls.Certificate, error) {
	// Generate key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}

	// Create serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Strip port from host
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host // No port
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: hostname,
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(1, 0, 0), // 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add SANs
	if ip := net.ParseIP(hostname); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{hostname}
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, tc.caCert, &key.PublicKey, tc.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create host certificate: %w", err)
	}

	// Create tls.Certificate
	cert := &tls.Certificate{
		Certificate: [][]byte{certDER, tc.caCert.Raw},
		PrivateKey:  key,
	}

	return cert, nil
}

// GetTLSConfig returns a tls.Config that uses dynamic certificate generation.
func (tc *TLSConfig) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return tc.GetCertForHost(hello.ServerName)
		},
	}
}

// CACertPath returns the path where CA cert should be stored.
func CACertPath() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "currier", "proxy", "ca.crt")
}

// CAKeyPath returns the path where CA key should be stored.
func CAKeyPath() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "currier", "proxy", "ca.key")
}

// ExportCACert copies the CA certificate to the specified path.
func (tc *TLSConfig) ExportCACert(destPath string) error {
	if tc.caCert == nil {
		return fmt.Errorf("no CA certificate loaded")
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: tc.caCert.Raw})
}

// CACertPEM returns the CA certificate in PEM format.
func (tc *TLSConfig) CACertPEM() []byte {
	if tc.caCert == nil {
		return nil
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tc.caCert.Raw})
}
