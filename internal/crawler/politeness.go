package crawler

import (
	"context"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the crawler when no --user-agent is configured
// and header rotation is off.
const DefaultUserAgent = "GoGoGoBot/1.0 (+https://github.com/BenjaminSRussell/go_go_go)"

// DefaultPerHostConcurrency caps simultaneous requests to a single host so a
// large --workers pool cannot hammer one site.
const DefaultPerHostConcurrency = 2

// robotsAgentToken returns the product token robots.txt groups are matched
// against, e.g. "GoGoGoBot/1.0 (+url)" -> "GoGoGoBot".
func robotsAgentToken(userAgent string) string {
	ua := strings.TrimSpace(userAgent)
	if ua == "" {
		ua = DefaultUserAgent
	}
	if i := strings.IndexAny(ua, "/ "); i > 0 {
		return ua[:i]
	}
	return ua
}

// hostGovernor limits per-host concurrency and spaces request start times by
// a per-host minimum delay (robots.txt Crawl-Delay).
type hostGovernor struct {
	limit int

	mu    sync.Mutex
	hosts map[string]*hostSlot
}

type hostSlot struct {
	sem  chan struct{}
	next time.Time // earliest start time for the next request
}

func newHostGovernor(limit int) *hostGovernor {
	if limit <= 0 {
		limit = DefaultPerHostConcurrency
	}
	return &hostGovernor{limit: limit, hosts: make(map[string]*hostSlot)}
}

func (g *hostGovernor) slot(host string) *hostSlot {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, ok := g.hosts[host]
	if !ok {
		s = &hostSlot{sem: make(chan struct{}, g.limit)}
		g.hosts[host] = s
	}
	return s
}

// acquire blocks until a request to host may start: a concurrency slot is free
// and at least delay has passed since the previous request start to that host.
// The returned release func must be called when the request finishes.
func (g *hostGovernor) acquire(ctx context.Context, host string, delay time.Duration) (func(), error) {
	s := g.slot(host)
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
	release := func() { <-s.sem }

	if delay > 0 {
		g.mu.Lock()
		now := time.Now()
		start := s.next
		if start.Before(now) {
			start = now
		}
		s.next = start.Add(delay)
		g.mu.Unlock()

		if wait := time.Until(start); wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-ctx.Done():
				release()
				return func() {}, ctx.Err()
			}
		}
	}
	return release, nil
}
