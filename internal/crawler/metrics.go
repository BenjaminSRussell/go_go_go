package crawler

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// scrapeMetrics holds Prometheus counters/gauges for an optional /metrics server.
type scrapeMetrics struct {
	pagesFetched  atomic.Int64
	status2xx     atomic.Int64
	status3xx     atomic.Int64
	status4xx     atomic.Int64
	status5xx     atomic.Int64
	retries       atomic.Int64
	activeWorkers atomic.Int64
	frontierSize  atomic.Int64
}

func (m *scrapeMetrics) observeStatus(code int) {
	switch {
	case code >= 200 && code < 300:
		m.status2xx.Add(1)
	case code >= 300 && code < 400:
		m.status3xx.Add(1)
	case code >= 400 && code < 500:
		m.status4xx.Add(1)
	case code >= 500:
		m.status5xx.Add(1)
	}
}

func (m *scrapeMetrics) handler(c *Crawler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c != nil && c.frontier != nil {
			m.frontierSize.Store(int64(c.frontier.Size()))
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintf(w, "# HELP gogogo_pages_fetched_total Pages successfully fetched\n")
		fmt.Fprintf(w, "# TYPE gogogo_pages_fetched_total counter\n")
		fmt.Fprintf(w, "gogogo_pages_fetched_total %d\n", m.pagesFetched.Load())
		fmt.Fprintf(w, "# HELP gogogo_http_responses_total HTTP responses by status class\n")
		fmt.Fprintf(w, "# TYPE gogogo_http_responses_total counter\n")
		fmt.Fprintf(w, "gogogo_http_responses_total{class=\"2xx\"} %d\n", m.status2xx.Load())
		fmt.Fprintf(w, "gogogo_http_responses_total{class=\"3xx\"} %d\n", m.status3xx.Load())
		fmt.Fprintf(w, "gogogo_http_responses_total{class=\"4xx\"} %d\n", m.status4xx.Load())
		fmt.Fprintf(w, "gogogo_http_responses_total{class=\"5xx\"} %d\n", m.status5xx.Load())
		fmt.Fprintf(w, "# HELP gogogo_retries_total Retry attempts\n")
		fmt.Fprintf(w, "# TYPE gogogo_retries_total counter\n")
		fmt.Fprintf(w, "gogogo_retries_total %d\n", m.retries.Load())
		fmt.Fprintf(w, "# HELP gogogo_frontier_size Pending URLs in frontier\n")
		fmt.Fprintf(w, "# TYPE gogogo_frontier_size gauge\n")
		fmt.Fprintf(w, "gogogo_frontier_size %d\n", m.frontierSize.Load())
		fmt.Fprintf(w, "# HELP gogogo_active_workers In-flight workers\n")
		fmt.Fprintf(w, "# TYPE gogogo_active_workers gauge\n")
		fmt.Fprintf(w, "gogogo_active_workers %d\n", m.activeWorkers.Load())
	}
}

func (c *Crawler) startMetricsServer() {
	if c.config.MetricsAddr == "" || c.metrics == nil {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", c.metrics.handler(c))
	addr := c.config.MetricsAddr
	go func() {
		fmt.Printf("[metrics] serving Prometheus text on http://%s/metrics\n", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("[metrics] server stopped: %v\n", err)
		}
	}()
}
