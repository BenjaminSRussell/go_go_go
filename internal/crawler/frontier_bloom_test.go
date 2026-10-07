package crawler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
)

func TestBloomRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seen.bloom")

	f := NewFrontier()
	f.Add(types.URLItem{URL: "https://example.com/a"})
	f.Add(types.URLItem{URL: "https://example.com/b"})
	if err := f.SaveBloom(path); err != nil {
		t.Fatal(err)
	}

	f2 := NewFrontier()
	if err := f2.LoadBloom(path); err != nil {
		t.Fatal(err)
	}
	if f2.Add(types.URLItem{URL: "https://example.com/a"}) {
		t.Fatal("expected seen URL to be rejected after load")
	}
	if !f2.Add(types.URLItem{URL: "https://example.com/c"}) {
		t.Fatal("expected new URL to be accepted")
	}
}

func TestBloomCorruptSoft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seen.bloom")
	if err := os.WriteFile(path, []byte("not-a-bloom"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := NewFrontier()
	if err := f.LoadBloom(path); err == nil {
		t.Fatal("expected corrupt bloom error")
	}
}
