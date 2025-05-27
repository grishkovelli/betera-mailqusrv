#!/bin/sh
set -e

# Database connection variables with default values
DB_HOST="${DB_HOST:-db}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-quadmin}"
DB_PASSWORD="${DB_PASSWORD:-quadmin}"
DB_NAME="${DB_NAME:-mailqu}"
SSL_MODE="${SSL_MODE:-disable}"
MIGRATIONS_PATH="${MIGRATIONS_PATH:-/migrations}"

if [ -n "$DB_HOST" ]; then
  until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER"; do
    echo "Waiting for database connection to $DB_HOST:$DB_PORT..."
    sleep 1
  done

  migrate -path "$MIGRATIONS_PATH" \
          -database "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=$SSL_MODE" \
          up
fi

exec "$@"