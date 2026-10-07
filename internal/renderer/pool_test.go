package renderer

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func fakeStart(counter *atomic.Int64, stopped *atomic.Int64) func() (context.Context, context.CancelFunc) {
	return func() (context.Context, context.CancelFunc) {
		counter.Add(1)
		ctx, cancel := context.WithCancel(context.Background())
		return ctx, func() { stopped.Add(1); cancel() }
	}
}

func TestTabPoolCapsConcurrency(t *testing.T) {
	var starts, stops atomic.Int64
	p := newTabPool(2, time.Hour, fakeStart(&starts, &stops))
	defer p.close()

	var cur, peak atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, release, err := p.acquire(context.Background())
			if err != nil {
				t.Error(err)
				return
			}
			n := cur.Add(1)
			for {
				old := peak.Load()
				if n <= old || peak.CompareAndSwap(old, n) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			cur.Add(-1)
			release(nil)
		}()
	}
	wg.Wait()
	if peak.Load() > 2 {
		t.Fatalf("peak concurrency %d exceeds cap 2", peak.Load())
	}
	if starts.Load() != 1 {
		t.Fatalf("browser should start once, started %d", starts.Load())
	}
	if s := p.stats(); s.Renders != 8 || s.InFlight != 0 {
		t.Fatalf("unexpected stats %+v", s)
	}
}

func TestTabPoolAcquireRespectsContext(t *testing.T) {
	var starts, stops atomic.Int64
	p := newTabPool(1, time.Hour, fakeStart(&starts, &stops))
	defer p.close()
	_, release, _ := p.acquire(context.Background())
	defer release(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, _, err := p.acquire(ctx); err == nil {
		t.Fatal("expected timeout while pool is full")
	}
}

func TestTabPoolStopsIdleBrowserAndRelaunches(t *testing.T) {
	var starts, stops atomic.Int64
	p := newTabPool(2, 20*time.Millisecond, fakeStart(&starts, &stops))
	defer p.close()

	_, release, _ := p.acquire(context.Background())
	release(nil)
	time.Sleep(80 * time.Millisecond)
	if stops.Load() != 1 || p.stats().IdleStops != 1 {
		t.Fatalf("idle browser not stopped: stops=%d stats=%+v", stops.Load(), p.stats())
	}

	_, release, _ = p.acquire(context.Background())
	release(nil)
	if starts.Load() != 2 {
		t.Fatalf("browser should relaunch on demand, starts=%d", starts.Load())
	}
}

func TestTabPoolCloseRejects(t *testing.T) {
	var starts, stops atomic.Int64
	p := newTabPool(1, time.Hour, fakeStart(&starts, &stops))
	p.close()
	if _, _, err := p.acquire(context.Background()); err == nil {
		t.Fatal("acquire after close should fail")
	}
}

func TestParseChromeFlag(t *testing.T) {
	cases := map[string]struct {
		name  string
		value interface{}
	}{
		"--disable-extensions":      {"disable-extensions", true},
		"window-size=1280,800":      {"window-size", "1280,800"},
		"--enable-automation=false": {"enable-automation", false},
	}
	for in, want := range cases {
		n, v := ParseChromeFlag(in)
		if n != want.name || v != want.value {
			t.Errorf("%q => %q,%v want %q,%v", in, n, v, want.name, want.value)
		}
	}
	if n, _ := ParseChromeFlag("  "); n != "" {
		t.Error("blank flag should be ignored")
	}
}
