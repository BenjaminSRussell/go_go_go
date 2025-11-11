# Go Go Go Scraper

A high-performance web crawler written in Go with advanced anti-bot evasion capabilities. Optimized for URL discovery, data extraction, and sitemap generation.

## Features

### Core Capabilities
- **Concurrent Crawling**: Configurable worker pools (default: 256 workers)
- **Smart URL Discovery**: Multiple seeding strategies (sitemap, Certificate Transparency, Common Crawl)
- **Politeness**: Per-host rate limiting with configurable delays
- **Deduplication**: Bloom filter-based URL deduplication (handles 100M+ URLs)
- **Resume Support**: Save and resume crawl state
- **Robot Exclusion**: Respects robots.txt by default
- **Sitemap Export**: Generate XML sitemaps from crawl results

### Advanced Features

**Ethical Use Warning**: These features are designed for legitimate web scraping with permission, security testing with authorization, or educational purposes. Always respect robots.txt and website terms of service.

- **TLS Fingerprinting**: Mimics real browser TLS handshakes (Chrome, Firefox, Safari, Edge) using `utls` to bypass JA3 fingerprint detection
- **Browser Header Rotation**: Rotates complete, matching browser header sets (User-Agent + 15+ additional headers) to appear as real users
- **JavaScript Rendering**: Headless Chrome integration with `chromedp` for SPA/React/Vue sites
- **Intelligent Retry Logic**: Exponential backoff with per-host tracking for 429/5xx errors
- **Advanced HTML Parsing**: CSS selector-based extraction with `goquery` for structured data
- **SQLite Storage**: Queryable database storage with full-text search and analytics
- **Persona Management**: Session-based crawling with behavioral delays and fingerprint consistency

## Installation

```bash
# Clone the repository
git clone https://github.com/BenjaminSRussell/go_go_go.git
cd go_go_go

# Build
go build -o gogogoscraper ./cmd/gogogoscraper
```

## Usage

### Basic Crawl

```bash
./gogogoscraper crawl --start-url https://example.com
```

### Advanced Crawl with Enhanced Features

```bash
# Enable all advanced features
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 64 \
  --timeout 15 \
  --enable-tls-fingerprint \
  --enable-js-rendering \
  --enable-sqlite \
  --use-header-rotation \
  --max-retries 5
```

### Focused Crawl (Respectful Mode)

```bash
# Conservative settings that respect rate limits
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 32 \
  --timeout 30 \
  --seeding-strategy sitemap \
  --max-retries 2
```

### Data Extraction Crawl

```bash
# Use SQLite for queryable data
./gogogoscraper crawl \
  --start-url https://example.com \
  --enable-sqlite \
  --enable-js-rendering \
  --seeding-strategy sitemap
```

## Command-Line Options

### Basic Options

| Flag | Default | Description |
|------|---------|-------------|
| `--start-url` | required | Starting URL |
| `--workers` | 256 | Number of concurrent workers |
| `--timeout` | 20 | Request timeout in seconds |
| `--data-dir` | ./data | Storage location |
| `--seeding-strategy` | all | URL discovery: none/sitemap/ct/commoncrawl/all |
| `--ignore-robots` | false | Skip robots.txt (not recommended) |
| `--max-retries` | 3 | Maximum retry attempts per URL |

### Advanced Options

| Flag | Default | Description |
|------|---------|-------------|
| `--enable-tls-fingerprint` | false | Mimic browser TLS fingerprints (bypass JA3 detection) |
| `--enable-js-rendering` | false | Render JavaScript with headless Chrome |
| `--enable-sqlite` | false | Use SQLite instead of JSONL for queryable data |
| `--use-header-rotation` | true | Rotate realistic browser headers |
| `--enable-personas` | false | Enable persona-based crawling with session persistence |
| `--enable-weighted-nav` | false | Use weighted navigation (prefer visible/important links) |

## Advanced Features Explained

### 1. TLS Fingerprinting

**Problem**: WAFs check your TLS fingerprint (JA3), not just User-Agent.

**Solution**: Uses `utls` to impersonate real browsers' TLS handshakes.

```bash
--enable-tls-fingerprint
```

Automatically rotates between Chrome, Firefox, Safari, and Edge fingerprints.

### 2. Browser Header Rotation

**Problem**: WAFs check 15+ headers, not just User-Agent.

**Solution**: Rotates complete, matching header sets:
- User-Agent
- Accept
- Accept-Language
- Accept-Encoding
- Sec-Ch-Ua (Chromium)
- Sec-Ch-Ua-Platform
- Sec-Fetch-* headers
- And more...

```bash
--use-header-rotation=true
```

TLS profile, headers, and User-Agent always match (e.g., Chrome TLS = Chrome headers).

### 3. JavaScript Rendering

**Problem**: SPA frameworks (React, Vue, Angular) render content client-side.

**Solution**: Hybrid escalation system:
1. Fast HTTP fetch first
2. Check if JS-heavy (< 500 bytes or has `<div id="root">`)
3. Escalate to headless Chrome pool
4. Render JS and extract final HTML

```bash
--enable-js-rendering
```

**Performance**: Only renders pages that need it (~5-10% of pages).

### 4. SQLite Storage

**Problem**: JSONL is write-only, can't query data.

**Solution**: SQLite with full schema:
- `pages` table: URL, status, title, content length, etc.
- `links` table: Source → target relationships
- `meta_tags` table: SEO metadata
- `structured_data` table: JSON-LD data

```bash
--enable-sqlite
```

Query examples:
```sql
-- Find all 404s
SELECT url FROM pages WHERE status_code = 404;

-- Most linked pages
SELECT target_url, COUNT(*) as count FROM links
GROUP BY target_url ORDER BY count DESC LIMIT 10;

-- Pages with specific meta tag
SELECT url FROM meta_tags WHERE name = 'og:type' AND content = 'article';
```

### 5. Intelligent Retry Logic

**Automatic features** (no flag needed):
- Exponential backoff: 1s → 2s → 4s → 8s
- Per-host tracking (doesn't slow down other hosts)
- Aggressive backoff for 429 (rate limit)
- Jitter (±20%) to avoid thundering herd

**Behavior**:
- 429: 2x backoff duration
- 5xx: Standard backoff
- Network errors: Retry with backoff
- 4xx (except 429): No retry

## Seeding Strategies

| Strategy | Description | Best For |
|----------|-------------|----------|
| `none` | Only start URL | Single page testing |
| `sitemap` | Discover from sitemap.xml | Fast, focused crawls |
| `ct` | Certificate Transparency logs | Subdomain discovery |
| `commoncrawl` | Query Common Crawl index | Historical URL discovery |
| `all` | Use all methods | Maximum coverage |

## Output Formats

### JSONL (Default)

Location: `./data/sitemap.jsonl`

```json
{"url":"https://example.com/","depth":0,"status_code":200,"content_length":1024,"title":"Example","link_count":5,"crawled_at":"2025-01-01T12:00:00Z"}
```

### SQLite (with `--enable-sqlite`)

Location: `./data/crawl.db`

Tables: `pages`, `links`, `meta_tags`, `structured_data`

Query with any SQLite client:
```bash
sqlite3 ./data/crawl.db "SELECT * FROM pages WHERE status_code = 200 LIMIT 10"
```

### XML Sitemap

```bash
./gogogoscraper export-sitemap --data-dir ./data --output sitemap.xml
```

## Performance & Recommendations

### Network-Bound Performance

Typical: 50-200 URLs/minute (depends on page size)

**Why slow?** Network I/O is the bottleneck:
- Page download: 700-900ms (70-90%)
- Network RTT: 50-150ms (10-20%)
- Processing: < 50ms (< 5%)

### Recommended Settings

```bash
# Fast focused crawl (respects servers)
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 64 \
  --timeout 10 \
  --seeding-strategy sitemap \
  --max-retries 2

# Maximum features (for authorized testing)
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 32 \
  --enable-tls-fingerprint \
  --enable-js-rendering \
  --enable-personas \
  --use-header-rotation \
  --max-retries 5

# Data extraction (queryable results)
./gogogoscraper crawl \
  --start-url https://example.com \
  --enable-sqlite \
  --enable-js-rendering \
  --seeding-strategy all
```

## Architecture

### Components

```
internal/
├── crawler/         # Main crawler engine with worker pools
├── parser/          # HTML parsing with goquery (CSS selectors)
├── http/            # Headers, TLS, retry logic
├── renderer/        # Headless Chrome (chromedp)
├── persona/         # Persona management for behavioral crawling
├── navigation/      # Weighted navigation strategies
├── seeding/         # URL discovery strategies
├── storage/         # JSONL and SQLite storage
└── types/           # Shared types
```

### Key Design Patterns

- **Worker Pool**: Semaphore-based concurrency control
- **Round-Robin Scheduling**: Fair host distribution
- **Bloom Filters**: Memory-efficient deduplication
- **Per-Host Backoff**: Intelligent retry without global slowdown
- **Hybrid Rendering**: Fast HTTP + selective JS rendering

## Ethical Use Guidelines

This tool includes powerful anti-bot features. Use responsibly:

**Allowed**:
- Scraping with explicit permission
- Authorized penetration testing
- Security research with consent
- Educational purposes
- Your own websites

**Not Allowed**:
- Bypassing rate limits without permission
- Ignoring robots.txt (without good reason)
- DDoS or mass targeting
- Scraping copyrighted content
- Violating terms of service

**Always**:
1. Respect `robots.txt` (use `--ignore-robots` sparingly)
2. Use reasonable worker counts (< 100 for most sites)
3. Honor rate limiting (don't disable retry backoff)
4. Identify your bot (User-Agent includes "GoGoGoBot")

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| Slow crawling | Normal - network I/O bound | Expected behavior |
| Many timeouts | Unreachable hosts from CT logs | Use `--seeding-strategy sitemap` |
| High memory | Too many workers × large pages | Reduce `--workers` to 32-64 |
| 429 errors | Rate limited | Reduce workers, increase delays |
| Empty results | JS-rendered content | Enable `--enable-js-rendering` |

## Comparison with Rust Scraper

### Advantages

- **Native Concurrency**: Goroutines are lighter than async tasks
- **Better Tooling**: Excellent HTML parsing and HTTP libraries
- **Simpler Architecture**: Less boilerplate than Rust
- **Faster Development**: Easier to add features
- **Anti-Bot Evasion**: More advanced fingerprinting

### Trade-offs

- Slightly higher memory usage (but still efficient)
- GC pauses (negligible for I/O-bound workload)

## Development

```bash
# Build
go build -o gogogoscraper ./cmd/gogogoscraper

# Test
go test ./...

# Run with race detector
go run -race ./cmd/gogogoscraper crawl --start-url https://example.com
```

## License

MIT License - Use responsibly

## Contributing

Contributions welcome! Please:
1. Respect ethical guidelines
2. Add tests for new features
3. Update documentation
4. Follow Go best practices

## Disclaimer

This tool is provided for educational and authorized testing purposes only. The authors are not responsible for misuse. Always obtain permission before scraping websites and respect robots.txt.
