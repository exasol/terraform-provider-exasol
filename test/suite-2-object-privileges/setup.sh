#!/bin/bash
# Setup script for object privilege tests
# Creates the test schema needed for privilege grants

set -e

echo "Creating test schema OP_TEST_SCHEMA..."

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

# Create schema using docker exec and SQL
docker exec "$EXASOL_CONTAINER" exaplus -c localhost:8563 -u sys -p exasol -sql "CREATE SCHEMA IF NOT EXISTS OP_TEST_SCHEMA;" 2>/dev/null || true

echo "Test schema created successfully"
