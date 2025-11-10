#!/bin/bash

set -e

GO_BIN="${GO_BIN:-/usr/local/go/bin/go}"
BINARY_NAME="gogogoscraper"

show_help() {
	cat <<EOF
Usage: ./run.sh [COMMAND] [OPTIONS]

Commands:
  crawl     Start a new crawl
  resume    Resume a previous crawl
  export    Export crawl results
  test      Run tests only
  help      Show this help message

Crawl Options:
  --url URL                  Starting URL (required)
  --workers N                Number of concurrent workers (default: 4)
  --timeout N                Request timeout in seconds (default: 30)
  --data-dir DIR             Directory for crawl data (default: ./crawl_data)
  --seeding-strategy STR     Seeding strategy: none, sitemap, ct, commoncrawl, all (default: sitemap)
  --ignore-robots            Ignore robots.txt
  --enable-proxies           Enable proxy rotation
  --enable-js-rendering      Enable JavaScript rendering
  --enable-personas          Enable user personas
  --enable-sqlite            Enable SQLite storage
  --max-personas N           Maximum personas (default: 50)
  --max-retries N            Maximum retry attempts (default: 3)

Examples:
  ./run.sh crawl --url https://example.com --workers 8 --data-dir ./my_crawl
  ./run.sh crawl --url https://example.com --enable-personas --enable-proxies
  ./run.sh test
  ./run.sh help
EOF
}

setup() {
	echo "📦 Tidying Go dependencies..."
	$GO_BIN mod tidy || { echo "❌ Failed to tidy dependencies"; exit 1; }
	
	echo "✓ Building $BINARY_NAME..."
	$GO_BIN build -o $BINARY_NAME ./cmd/gogogoscraper || { echo "❌ Failed to build"; exit 1; }
	echo "✓ Build successful"
}

run_tests() {
	echo "🧪 Running tests..."
	$GO_BIN test ./... || { echo "❌ Tests failed"; exit 1; }
	echo "✓ All tests passed"
}

run_crawler() {
	if [ $# -eq 0 ]; then
		echo "❌ Error: No command specified"
		show_help
		exit 1
	fi
	
	setup
	
	echo "🚀 Starting crawler..."
	./$BINARY_NAME "$@"
}

main() {
	case "${1:-help}" in
		help)
			show_help
			;;
		test)
			setup
			run_tests
			;;
		crawl|resume|export)
			run_crawler "$@"
			;;
		*)
			echo "❌ Unknown command: $1"
			show_help
			exit 1
			;;
	esac
}

main "$@"