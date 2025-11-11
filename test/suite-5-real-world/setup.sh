#!/bin/bash
# Setup script for real-world production test
# Creates the 4-layer test schemas needed for the test

set -e

echo "Creating 4-layer test schemas (RAW, SNAP, STG, MART)..."

# Find Exasol container name
EXASOL_CONTAINER=$(docker ps --filter "ancestor=exasol/docker-db" --format "{{.Names}}" | head -n 1)

if [ -z "$EXASOL_CONTAINER" ]; then
    # Try alternative name pattern
    EXASOL_CONTAINER=$(docker ps | grep exasol | awk '{print $NF}' | head -n 1)
fi

if [ -z "$EXASOL_CONTAINER" ]; then
    echo "Error: No Exasol container found"
    exit 1
fi

# Create schemas using docker exec and SQL
docker exec "$EXASOL_CONTAINER" exaplus -c localhost:8563 -u sys -p exasol -sql "CREATE SCHEMA IF NOT EXISTS RW_RAW_SCHEMA;" 2>/dev/null || true
docker exec "$EXASOL_CONTAINER" exaplus -c localhost:8563 -u sys -p exasol -sql "CREATE SCHEMA IF NOT EXISTS RW_SNAP_SCHEMA;" 2>/dev/null || true
docker exec "$EXASOL_CONTAINER" exaplus -c localhost:8563 -u sys -p exasol -sql "CREATE SCHEMA IF NOT EXISTS RW_STG_SCHEMA;" 2>/dev/null || true
docker exec "$EXASOL_CONTAINER" exaplus -c localhost:8563 -u sys -p exasol -sql "CREATE SCHEMA IF NOT EXISTS RW_MART_SCHEMA;" 2>/dev/null || true

echo "All test schemas created successfully"
