#!/bin/bash
# download_geoip.sh — Download ip2region GeoIP database files
# Usage: ./scripts/download_geoip.sh
#
# Downloads ip2region xdb files via sparse checkout to minimize bandwidth.
# Run this before local development if you haven't built via Docker.

set -e

DATA_DIR="data"
REPO_URL="https://github.com/lionsoul2014/ip2region.git"
TMP_DIR=$(mktemp -d)

echo "📦 Downloading ip2region data files..."

mkdir -p "$DATA_DIR"

# Sparse checkout — only clone the data directory
git clone --depth 1 --filter=blob:none --sparse "$REPO_URL" "$TMP_DIR" 2>/dev/null
cd "$TMP_DIR"
git sparse-checkout set data
cd -

# Copy xdb files
cp "$TMP_DIR/data/ip2region_v4.xdb" "$DATA_DIR/ip2region_v4.xdb"
cp "$TMP_DIR/data/ip2region_v6.xdb" "$DATA_DIR/ip2region_v6.xdb"

# Cleanup
rm -rf "$TMP_DIR"

echo "✅ GeoIP data downloaded:"
ls -lh "$DATA_DIR"/*.xdb
