package renderer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Defaults for the Chrome tab pool (#10).
const (
	DefaultMaxTabs     = 4
	DefaultIdleTimeout = 60 * time.Second
)

// Config controls headless Chrome resource use.
type Config struct {
	// MaxTabs caps concurrent Chrome targets (tabs). <= 0 uses DefaultMaxTabs.
	MaxTabs int
	// IdleTimeout shuts the browser down after this long with no renders;
	// it is relaunched lazily on the next Render. <= 0 uses DefaultIdleTimeout.
	IdleTimeout time.Duration
	// DisableGPU passes --disable-gpu (default true via DefaultConfig).
	DisableGPU bool
	// ExtraFlags are additional Chrome switches, "name" or "name=value".
	ExtraFlags []string
}

// DefaultConfig returns the recommended configuration.
func DefaultConfig() Config {
	return Config{MaxTabs: DefaultMaxTabs, IdleTimeout: DefaultIdleTimeout, DisableGPU: true}
}

func (c Config) normalized() Config {
	if c.MaxTabs <= 0 {
		c.MaxTabs = DefaultMaxTabs
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
	return c
}

// Stats is a snapshot of renderer activity.
type Stats struct {
	Renders      int64 // completed render attempts
	Failures     int64 // render attempts that returned an error
	InFlight     int64 // tabs currently open
	BrowserStart int64 // times Chrome was (re)launched
	IdleStops    int64 // times Chrome was shut down for idleness
}

// tabPool caps concurrent tabs and tears the browser down when idle. It is
// independent of chromedp so its behavior can be unit-tested.
type tabPool struct {
	sem         chan struct{}
	idleTimeout time.Duration
	start       func() (context.Context, context.CancelFunc)

	mu        sync.Mutex
	browser   context.Context
	stop      context.CancelFunc
	idleTimer *time.Timer
	closed    bool

	renders, failures, inFlight, starts, idleStops atomic.Int64
}

func newTabPool(maxTabs int, idle time.Duration, start func() (context.Context, context.CancelFunc)) *tabPool {
	return &tabPool{sem: make(chan struct{}, maxTabs), idleTimeout: idle, start: start}
}

// acquire blocks until a tab slot is free (or ctx is done) and returns the
// browser context to open the tab in plus a release func.
func (p *tabPool) acquire(ctx context.Context) (context.Context, func(err error), error) {
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		<-p.sem
		return nil, nil, context.Canceled
	}
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer = nil
	}
	if p.browser == nil {
		p.browser, p.stop = p.start()
		p.starts.Add(1)
	}
	browser := p.browser
	p.mu.Unlock()
	p.inFlight.Add(1)

	var once sync.Once
	release := func(err error) {
		once.Do(func() {
			p.renders.Add(1)
			if err != nil {
				p.failures.Add(1)
			}
			if p.inFlight.Add(-1) == 0 {
				p.armIdle()
			}
			<-p.sem
		})
	}
	return browser, release, nil
}

func (p *tabPool) armIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.browser == nil {
		return
	}
	if p.idleTimer != nil {
		p.idleTimer.Stop()
	}
	p.idleTimer = time.AfterFunc(p.idleTimeout, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.inFlight.Load() == 0 && p.browser != nil {
			p.stop()
			p.browser, p.stop = nil, nil
			p.idleStops.Add(1)
		}
		p.idleTimer = nil
	})
}

func (p *tabPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer = nil
	}
	if p.stop != nil {
		p.stop()
		p.browser, p.stop = nil, nil
	}
}

func (p *tabPool) stats() Stats {
	return Stats{
		Renders:      p.renders.Load(),
		Failures:     p.failures.Load(),
		InFlight:     p.inFlight.Load(),
		BrowserStart: p.starts.Load(),
		IdleStops:    p.idleStops.Load(),
	}
}
