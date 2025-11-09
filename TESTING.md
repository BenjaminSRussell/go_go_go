# Testing & Validation Guide

## Current Status

✅ **Code Structure**: All components implemented
✅ **Error Handling**: Comprehensive nil checks and panic recovery
✅ **Graceful Degradation**: Features fail gracefully if dependencies unavailable
⚠️ **Dependencies**: Requires network access to download (PuerkitoBio/goquery, etc.)
⚠️ **Compilation**: Needs `go mod tidy` with network access

## Known Issues & Mitigations

### 1. Dependency Downloads
**Issue**: go.sum missing entries for new dependencies
**Mitigation**: Run `go mod tidy` with internet access
**Fallback**: Standard crawler still works without advanced features

### 2. Nil Pointer Dereferences
**Handled**: All critical components have nil checks in:
- `internal/crawler/safeguards.go`: ValidateEnhancedCrawler()
- `internal/crawler/enhanced.go`: Nil checks on all component initialization
- `internal/crawler/safeguards.go`: SafeProcessor with panic recovery

### 3. Race Conditions
**Handled**:
- All counters use `atomic.Int64`
- All maps protected with `sync.RWMutex`
- Persona pool has per-persona locks
- Proxy manager has fine-grained locking

### 4. Resource Leaks
**Handled**:
- `SafeClose()` function closes all resources with panic recovery
- `defer crawler.Close()` in CLI
- Separate cleanup goroutine for periodic maintenance

### 5. Network Failures
**Handled**:
- Retry handler with exponential backoff
- Per-host tracking prevents cascade failures
- Non-fatal errors log warnings and continue

### 6. Feature Initialization Failures
**Handled**:
- Proxy manager: Logs warning, continues without proxies
- Chrome renderer: Logs warning, continues without JS rendering
- SQLite: Logs warning, falls back to JSONL

## Testing Checklist

### Unit Tests (To Be Added)
- [ ] Persona pool creation and lifecycle
- [ ] Proxy validation and leasing
- [ ] Weighted link selection
- [ ] Log-normal distribution generation
- [ ] Retry logic with backoff

### Integration Tests (To Be Added)
- [ ] Standard crawler without advanced features
- [ ] Enhanced crawler with all features enabled
- [ ] Enhanced crawler with partial features
- [ ] Error injection and recovery

### Edge Cases Handled

#### Empty/Nil Inputs
- ✅ Nil crawler in SafeClose()
- ✅ Nil response in processURLEnhanced()
- ✅ Empty body in HTML parsing
- ✅ Invalid URLs in frontier

#### Configuration Extremes
- ✅ Workers = 0 (validation rejects)
- ✅ Workers > 1000 (validation rejects)
- ✅ Timeout = 0 (validation rejects)
- ✅ Max retries < 0 (validation rejects)

#### Concurrency Issues
- ✅ Simultaneous persona access (mutex protected)
- ✅ Proxy lease conflicts (atomic operations)
- ✅ Frontier access (RWMutex protected)
- ✅ Statistics updates (atomic counters)

#### Resource Exhaustion
- ✅ Semaphore limits concurrent workers
- ✅ Persona pool has max size
- ✅ Proxy lease has expiration
- ✅ Connection pooling with limits

## Manual Testing Commands

### 1. Build Test
```bash
# Should list all errors
go build -o gogogoscraper ./cmd/gogogoscraper

# With dependencies resolved
go mod tidy && go build -o gogogoscraper ./cmd/gogogoscraper
```

### 2. Race Detector
```bash
go build -race -o gogogoscraper-race ./cmd/gogogoscraper
./gogogoscraper-race crawl --start-url https://example.com --workers 16
```

### 3. Standard Crawl (No Advanced Features)
```bash
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 32 \
  --timeout 15 \
  --seeding-strategy sitemap
```

### 4. Enhanced Crawl (All Features)
```bash
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 16 \
  --enable-personas \
  --enable-proxies \
  --enable-weighted-nav \
  --enable-sqlite \
  --max-personas 25
```

### 5. Stress Test
```bash
# High concurrency
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 256 \
  --enable-personas \
  --max-personas 100

# Long running
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 64 \
  --persona-lifetime 60 \
  --seeding-strategy all
```

### 6. Failure Injection
```bash
# Invalid URL
./gogogoscraper crawl --start-url "not-a-url"

# Unreachable host
./gogogoscraper crawl --start-url http://192.0.2.1

# With retries
./gogogoscraper crawl \
  --start-url http://httpstat.us/500 \
  --max-retries 5
```

## Expected Behavior

### Graceful Degradation
1. Proxy manager fails → Continue without proxies
2. Chrome fails → Continue without JS rendering
3. SQLite fails → Fall back to JSONL
4. Seeding fails → Continue with start URL only

### Error Recovery
1. Panic in URL processing → Caught, logged, marked as error
2. Network timeout → Retry with backoff
3. 429 rate limit → Aggressive backoff (2x)
4. Invalid HTML → Return empty links, continue

### Resource Cleanup
1. Context cancellation → All workers stop
2. Frontier empty → Graceful shutdown
3. SIGINT → Resources cleaned up via defer
4. Panic → Resources cleaned up in SafeClose()

## Performance Expectations

### Standard Mode
- 50-200 URLs/min (network bound)
- Memory: ~100-500MB
- CPU: 5-20% (mostly I/O wait)

### Enhanced Mode (All Features)
- 30-150 URLs/min (JS rendering + proxies)
- Memory: ~500MB-2GB (Chrome + persona pool)
- CPU: 20-60% (Chrome rendering)

## Security Considerations

### Handled
- ✅ SQL injection (using prepared statements)
- ✅ Path traversal (dataDir validation)
- ✅ DOS (worker semaphore limit)
- ✅ Resource exhaustion (timeouts, limits)

### User Responsibility
- ⚠️ Respect robots.txt
- ⚠️ Obtain authorization
- ⚠️ Use reasonable worker counts
- ⚠️ Monitor resource usage

## Next Steps for Production

1. **Add Unit Tests**: Cover all critical functions
2. **Add Integration Tests**: Test feature combinations
3. **Add Benchmarks**: Measure performance regressions
4. **Add Metrics**: Prometheus/statsd integration
5. **Add Circuit Breakers**: Fail fast on persistent errors
6. **Add Health Checks**: Monitor component health
7. **Add Distributed Tracing**: Debug complex issues
8. **Add Configuration Validation**: More comprehensive checks

## Debug Mode

To enable verbose logging, modify code to add:
```go
// In crawler.go
var DebugMode = os.Getenv("DEBUG") == "true"

// Throughout code
if DebugMode {
    log.Printf("[DEBUG] Persona: %v", persona.GetStats())
}
```

Then run:
```bash
DEBUG=true ./gogogoscraper crawl --start-url https://example.com
```
