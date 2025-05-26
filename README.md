For local deployment, simply run `docker-compose up`. After execution, the database will be created and migrations will be performed.
Once both containers are running, to simulate sending messages, you can run the command `docker exec <APP_CONTAINER_NAME> /app/mailer` which
will create `50` requests with different email addresses.

Features:
  - Statistics of processed messages GET /emails?status = pending | sent | failed
  - PK-based pagination to reduce load GET /emails
  - Additional goroutine that checks for stuck messages (if worker crashed) in `processing` status and changes their status to `pending` for subsequent processing
  - Configuration via `.env`
  - Retry sending messages with `failed` status
  - Log output
  - Unit tests for `handlers` and `worker`
  - Docker + docker-compose

To get statistics:

  ```
    # without pagination. Output up to 50 records (limited by SERVER_PAGE_SIZE)
    curl 'http://localhost:3000/emails?status=sent'

    # basic pagination by primary key
    curl 'http://localhost:3000/emails?status=sent&cursor=20'
  ```

For manual request sending:

  ```
    curl -H 'Content-Type: application/json' \
    -d '{ "to_address":"admin@mail.com","subject":"golang", "body": "Go probably the best language, u know?"}' \
    -X POST \
    http://localhost:3000/send-email
  ```

For testing run `go test ./internal/... -v`

Demonstration of `docker-compose`, `mailer`, and `httpie`.

<img src="screenshot.png" width="720">