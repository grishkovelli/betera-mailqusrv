# Build stage
FROM golang:1.24.1-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN cd ./cmd/server && \
  CGO_ENABLED=0 GOOS=linux go build -o /app/server
RUN cd ./cmd/mailer && \
  CGO_ENABLED=0 GOOS=linux go build -o /app/mailer

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl postgresql-client && \
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.3/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/mailer .
COPY --from=builder /app/.env ./.env

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

CMD ["/app/server"]