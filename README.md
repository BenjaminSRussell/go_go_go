# Go Go Go Scraper

A high-performance web crawler written in Go, optimized for URL discovery and sitemap generation. Built with concurrency, politeness, and efficiency in mind.

## Features

- **Concurrent Crawling**: Configurable worker pools (default: 256 workers)
- **Smart URL Discovery**: Multiple seeding strategies (sitemap, Certificate Transparency, Common Crawl)
- **Politeness**: Per-host rate limiting with configurable delays
- **Deduplication**: Bloom filter-based URL deduplication for memory efficiency
- **Resume Support**: Save and resume crawl state
- **Robot Exclusion**: Respects robots.txt (optional)
- **Sitemap Export**: Generate XML sitemaps from crawl results
- **Connection Pooling**: Optimized HTTP client with persistent connections

## Installation

```bash
go build -o gogogoscraper ./cmd/gogogoscraper
```

Or install directly:

```bash
go install github.com/BenjaminSRussell/go_go_go/cmd/gogogoscraper@latest
```

## Usage

### Basic Crawl

```bash
./gogogoscraper crawl --start-url https://example.com
```

### With Options

```bash
./gogogoscraper crawl \
  --start-url https://example.com \
  --workers 128 \
  --timeout 10 \
  --seeding-strategy sitemap
```

### Resume a Previous Crawl

```bash
./gogogoscraper resume --data-dir ./data
```

### Export Sitemap

```bash
./gogogoscraper export-sitemap \
  --data-dir ./data \
  --output sitemap.xml
```

## Command-Line Options

### `crawl` Command

| Flag | Default | Description |
|------|---------|-------------|
| `--start-url` | required | Starting URL |
| `--workers` | 256 | Number of concurrent workers |
| `--timeout` | 20 | Request timeout in seconds |
| `--data-dir` | ./data | Storage location |
| `--seeding-strategy` | all | Seeding strategy (see below) |
| `--ignore-robots` | false | Skip robots.txt |
| `--enable-redis` | false | Distributed mode (future) |
| `--redis-url` | - | Redis connection |

### `resume` Command

| Flag | Default | Description |
|------|---------|-------------|
| `--data-dir` | ./data | Storage location |

### `export-sitemap` Command

| Flag | Default | Description |
|------|---------|-------------|
| `--data-dir` | ./data | Storage location |
| `--output` | sitemap.xml | Output file path |
| `--include-lastmod` | true | Include lastmod in sitemap |
| `--include-changefreq` | true | Include changefreq in sitemap |
| `--default-priority` | 0.5 | Default priority value |

## Seeding Strategies

The scraper supports multiple URL discovery strategies:

- **none**: Only crawl from start URL
- **sitemap**: Discover URLs from sitemap.xml
- **ct**: Find subdomains via Certificate Transparency logs
- **commoncrawl**: Query Common Crawl index for known URLs
- **all**: Use all methods (default)

### Examples

```bash
# Focused crawl (skip subdomain discovery)
./gogogoscraper crawl --start-url https://example.com --seeding-strategy sitemap

# University sites (avoid internal hosts)
./gogogoscraper crawl --start-url https://www.university.edu --timeout 5 --seeding-strategy sitemap

# Maximum discovery
./gogogoscraper crawl --start-url https://example.com --workers 256 --seeding-strategy all
```

## Output

### JSONL Format

Crawl results are automatically saved to `./data/sitemap.jsonl`:

```json
{"url":"https://example.com/","depth":0,"status_code":200,"content_length":1024,"title":"Example","link_count":5,"crawled_at":"2025-01-01T12:00:00Z"}
```

### XML Sitemap

Export to XML format:

```bash
./gogogoscraper export-sitemap --data-dir ./data --output sitemap.xml
```

## Performance

### Optimizations

- **Connection Pooling**: Reuses HTTP connections with configurable idle connection limits
- **Bloom Filters**: Memory-efficient URL deduplication (100M URLs with 1% false positive rate)
- **Per-Host Politeness**: 100ms minimum delay between requests to the same host
- **Concurrent Workers**: Semaphore-based worker pool with configurable concurrency
- **Context Cancellation**: Graceful shutdown support

### Recommended Settings

```bash
# Fast focused crawl
./gogogoscraper crawl --start-url https://example.com --timeout 10 --workers 128 --seeding-strategy sitemap

# Deep crawl with all discovery methods
./gogogoscraper crawl --start-url https://example.com --workers 256 --timeout 20 --seeding-strategy all

# Conservative crawl (respect rate limits)
./gogogoscraper crawl --start-url https://example.com --workers 32 --timeout 30
```

## Architecture

### Components

- **Frontier**: Sharded URL queues with bloom filter deduplication and per-host politeness
- **Storage**: JSONL-based persistence with config and state management
- **Parser**: HTML link extraction with URL normalization and tracking parameter removal
- **Seeding**: Multiple discovery strategies for comprehensive URL gathering
- **Crawler**: Main engine with worker pool, context management, and robots.txt support

### Design Patterns

- Worker pool with semaphore-based concurrency control
- Round-robin host scheduling for fairness
- Atomic counters for thread-safe statistics
- Context-based cancellation for graceful shutdown

## Comparison with Rust Scraper

### Optimizations for Go

1. **Better Connection Pooling**: Go's `http.Transport` provides excellent connection reuse
2. **Goroutines**: Lightweight concurrency with easy-to-manage worker pools
3. **Channels & Semaphores**: Native concurrency primitives for backpressure
4. **Standard Library**: Robust HTML parsing and HTTP client out of the box

### Improvements

- Simplified architecture leveraging Go's concurrency model
- Better memory efficiency with bloom filters
- More flexible seeding strategies
- Cleaner separation of concerns

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| Slow crawling | Network I/O bound (expected for large pages) | Normal behavior |
| Many timeouts | Unreachable hosts from CT log discovery | Reduce timeout: `--timeout 5` or use `--seeding-strategy sitemap` |
| High memory usage | Too many concurrent large pages | Reduce workers: `--workers 64` |
| Stops unexpectedly | Frontier exhausted (all URLs processed) | Check `sitemap.jsonl` for results |

## Development

### Project Structure

```
go_go_go/
├── cmd/
│   └── gogogoscraper/    # Main CLI entry point
├── internal/
│   ├── cli/              # Cobra CLI commands
│   ├── crawler/          # Core crawler engine
│   ├── export/           # Sitemap export
│   ├── parser/           # HTML/URL parsing
│   ├── seeding/          # URL discovery strategies
│   ├── storage/          # Data persistence
│   └── types/            # Shared types
└── README.md
```

### Building

```bash
go build -o gogogoscraper ./cmd/gogogoscraper
```

### Testing

```bash
go test ./...
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions welcome! Please open an issue or submit a pull request.
