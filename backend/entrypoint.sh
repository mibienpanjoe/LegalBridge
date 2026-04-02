#!/bin/sh
set -e

echo "Running database migrations..."
/migrate

echo "Starting server..."
exec /server
