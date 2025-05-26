#!/bin/sh

set -e

if [ -n db ]; then
  until pg_isready -h db -p 5432 -U quadmin; do
    echo "Waiting for database connection..."
    sleep 1
  done

  migrate -path /migrations -database "postgres://quadmin:quadmin@db:5432/mailqu?sslmode=disable" up
fi

exec "$@"