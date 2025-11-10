package persona

import (
	"net/http"
	"testing"
	"time"
)

func TestPersonaPoolNew(t *testing.T) {
	pool := NewPersonaPool(10, 5*time.Minute, 50)

	if pool == nil {
		t.Error("Expected PersonaPool to be created")
	}
}

func TestPersonaPoolGetOrCreatePersona(t *testing.T) {
	pool := NewPersonaPool(10, 5*time.Minute, 50)

	persona1, err := pool.GetOrCreatePersona("example.com")
	if err != nil {
		t.Fatalf("Failed to get persona: %v", err)
	}

	if persona1 == nil {
		t.Error("Expected persona to be created")
	}

	persona2, err := pool.GetOrCreatePersona("example.com")
	if err != nil {
		t.Fatalf("Failed to get persona: %v", err)
	}

	if persona1.ID != persona2.ID {
		t.Error("Expected same persona for same domain")
	}
}

func TestPersonaNew(t *testing.T) {
	persona := NewPersona()

	if persona == nil {
		t.Error("Expected Persona to be created")
	}

	if persona.ID == "" {
		t.Error("Expected persona ID to be set")
	}
}

func TestPersonaGetThinkTime(t *testing.T) {
	persona := NewPersona()

	thinkTime := persona.GetThinkTime()

	if thinkTime < 0 {
		t.Errorf("Expected positive think time, got %v", thinkTime)
	}
}

func TestPersonaGetPageLoadTimeout(t *testing.T) {
	persona := NewPersona()

	timeout := persona.GetPageLoadTimeout()

	if timeout <= 0 {
		t.Errorf("Expected positive timeout, got %v", timeout)
	}
}

func TestPersonaAssignProxy(t *testing.T) {
	persona := NewPersona()

	proxyURL := "http://proxy.example.com:8080"
	persona.AssignProxy(proxyURL)

	if persona.ProxyURL != proxyURL {
		t.Errorf("Expected proxy URL %s, got %s", proxyURL, persona.ProxyURL)
	}
}

func TestPersonaShouldFollowLink(t *testing.T) {
	persona := NewPersona()

	weight := 0.8
	shouldFollow := persona.ShouldFollowLink(weight)

	if shouldFollow && weight < 0.5 {
		t.Error("Should not follow low-weight links")
	}
}

func TestPersonaApplyToRequest(t *testing.T) {
	persona := NewPersona()

	req, _ := http.NewRequest("GET", "https://example.com", nil)

	persona.ApplyToRequest(req)

	if req.Header.Get("User-Agent") == "" {
		t.Error("Expected User-Agent header to be set")
	}
}
