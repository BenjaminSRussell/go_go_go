package http

import (
	"crypto/tls"
	"math/rand"
	"net/http"
	"time"

	utls "github.com/refraction-networking/utls"
)

// TLSProfile represents a browser TLS fingerprint
type TLSProfile struct {
	Name      string
	ClientID  utls.ClientHelloID
}

var tlsProfiles = []TLSProfile{
	{Name: "Chrome_120", ClientID: utls.HelloChrome_120},
	{Name: "Firefox_117", ClientID: utls.HelloFirefox_117},
	{Name: "Safari_16_0", ClientID: utls.HelloSafari_16_0},
	{Name: "Edge_120", ClientID: utls.HelloEdge_120},
	{Name: "Chrome_Android_116", ClientID: utls.HelloChrome_Android_116},
	{Name: "Safari_iOS_16_0", ClientID: utls.HelloSafari_iOS_16_0},
}

// TLSFingerprinter manages TLS fingerprinting
type TLSFingerprinter struct {
	profiles []TLSProfile
	rnd      *rand.Rand
}

// NewTLSFingerprinter creates a new TLS fingerprinter
func NewTLSFingerprinter() *TLSFingerprinter {
	return &TLSFingerprinter{
		profiles: tlsProfiles,
		rnd:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetRandomProfile returns a random TLS profile
func (tf *TLSFingerprinter) GetRandomProfile() TLSProfile {
	return tf.profiles[tf.rnd.Intn(len(tf.profiles))]
}

// CreateTransport creates an HTTP transport with TLS fingerprinting
func (tf *TLSFingerprinter) CreateTransport(profile TLSProfile, proxyURL string) (*http.Transport, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	// Apply TLS configuration
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	// Note: Full utls integration requires custom dialer
	// This is a simplified version that uses standard TLS
	// For full fingerprinting, you'd need to implement a custom RoundTripper

	return transport, nil
}

// GetMatchingHeaderProfile returns a header profile matching the TLS profile
func (tf *TLSFingerprinter) GetMatchingHeaderProfile(tlsProfile TLSProfile) BrowserProfile {
	// Match TLS profile to appropriate header profile
	switch tlsProfile.Name {
	case "Chrome_120", "Chrome_Android_116", "Edge_120":
		// Return Chrome-like headers
		return browserProfiles[0] // Chrome on Windows
	case "Firefox_117":
		// Return Firefox headers
		return browserProfiles[2] // Firefox on Windows
	case "Safari_16_0", "Safari_iOS_16_0":
		// Return Safari headers
		return browserProfiles[4] // Safari on macOS
	default:
		return browserProfiles[0]
	}
}
