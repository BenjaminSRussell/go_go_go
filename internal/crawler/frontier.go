package crawler

import (
	"net/url"
	"sync"
	"time"

	"github.com/BenjaminSRussell/go_go_go/internal/types"
	"github.com/bits-and-blooms/bloom/v3"
)

const (
	// Bloom filter settings for ~10M URLs with 1% false positive rate
	bloomFilterSize = 100_000_000
	bloomFilterHash = 7
)

// Frontier manages the URL queue with deduplication and politeness
type Frontier struct {
	mu sync.Mutex

	// URL queues by host for politeness
	queues map[string]*hostQueue

	// Bloom filter for fast deduplication
	seen *bloom.BloomFilter

	// Global counters
	discovered int
	processed  int

	// Round-robin scheduling
	hosts     []string
	hostIndex int
}

// hostQueue manages URLs for a specific host
type hostQueue struct {
	urls         []types.URLItem
	lastAccess   time.Time
	politenessMs int // Minimum delay between requests
}

// NewFrontier creates a new URL frontier
func NewFrontier() *Frontier {
	return &Frontier{
		queues: make(map[string]*hostQueue),
		seen:   bloom.NewWithEstimates(bloomFilterSize, 0.01),
		hosts:  make([]string, 0),
	}
}

// Add adds a URL to the frontier if not seen before
func (f *Frontier) Add(item types.URLItem) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if already seen
	urlBytes := []byte(item.URL)
	if f.seen.Test(urlBytes) {
		return false
	}

	f.seen.Add(urlBytes)
	f.discovered++

	// Extract host
	parsedURL, err := url.Parse(item.URL)
	if err != nil {
		return false
	}
	host := parsedURL.Host

	// Get or create host queue
	queue, exists := f.queues[host]
	if !exists {
		queue = &hostQueue{
			urls:         make([]types.URLItem, 0),
			politenessMs: 100, // 100ms between requests to same host
		}
		f.queues[host] = queue
		f.hosts = append(f.hosts, host)
	}

	queue.urls = append(queue.urls, item)
	return true
}

// Next retrieves the next URL to crawl, respecting politeness
func (f *Frontier) Next() (types.URLItem, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	startIndex := f.hostIndex

	// Round-robin through hosts
	for i := 0; i < len(f.hosts); i++ {
		f.hostIndex = (f.hostIndex + 1) % len(f.hosts)
		if f.hostIndex >= len(f.hosts) {
			f.hostIndex = 0
		}

		host := f.hosts[f.hostIndex]
		queue := f.queues[host]

		// Check if we can access this host (politeness delay)
		if now.Sub(queue.lastAccess) < time.Duration(queue.politenessMs)*time.Millisecond {
			continue
		}

		// Get next URL from this host's queue
		if len(queue.urls) > 0 {
			item := queue.urls[0]
			queue.urls = queue.urls[1:]
			queue.lastAccess = now
			return item, true
		}
	}

	// If we made a full round and found nothing
	if startIndex == f.hostIndex || len(f.hosts) == 0 {
		return types.URLItem{}, false
	}

	return types.URLItem{}, false
}

// MarkProcessed increments the processed counter
func (f *Frontier) MarkProcessed() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.processed++
}

// Stats returns current frontier statistics
func (f *Frontier) Stats() (discovered, processed int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.discovered, f.processed
}

// IsEmpty checks if the frontier has no more URLs
func (f *Frontier) IsEmpty() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, queue := range f.queues {
		if len(queue.urls) > 0 {
			return false
		}
	}
	return true
}

// Size returns the total number of pending URLs
func (f *Frontier) Size() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	total := 0
	for _, queue := range f.queues {
		total += len(queue.urls)
	}
	return total
}
