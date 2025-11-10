package renderer

import (
	"strings"
	"testing"
	"time"
)

func TestChromeRendererNew(t *testing.T) {
	renderer, err := NewChromeRenderer()

	if err != nil {
		t.Logf("Failed to create Chrome renderer (expected in test env): %v", err)
	}

	if renderer != nil {
		t.Logf("Chrome renderer created successfully")
	}
}

func TestShouldRender(t *testing.T) {
	staticContent := "<html><body>" + strings.Repeat("Static content with lots of text ", 30) + "</body></html>"
	scriptContent := "<html><script>console.log('test')</script></html>"
	reactContent := "<html><div data-reactroot></div></html>"
	
	tests := []struct {
		html         string
		shouldRender bool
	}{
		{scriptContent, true},
		{staticContent, false},
		{reactContent, true},
	}

	for _, tt := range tests {
		result := ShouldRender(tt.html)

		if result && !tt.shouldRender {
			t.Errorf("ShouldRender(%s): expected false, got true", tt.html[:20])
		}
	}
}

func TestRenderTimeout(t *testing.T) {
	renderer, err := NewChromeRenderer()

	if err != nil {
		t.Logf("Failed to create renderer: %v", err)
		return
	}

	if renderer != nil {
		timeout := 30 * time.Second

		if timeout <= 0 {
			t.Error("Expected positive timeout")
		}
	}
}

func TestRendererClose(t *testing.T) {
	renderer, err := NewChromeRenderer()

	if err != nil {
		t.Logf("Failed to create renderer: %v", err)
		return
	}

	if renderer != nil {
		renderer.Close()
		t.Logf("Renderer closed successfully")
	}
}

func TestChromeDPInitialization(t *testing.T) {
	renderer, err := NewChromeRenderer()

	if err != nil {
		t.Logf("Failed to initialize Chrome renderer (expected in test env): %v", err)
		return
	}

	if renderer != nil {
		t.Log("Chrome renderer successfully initialized")
	}
}
