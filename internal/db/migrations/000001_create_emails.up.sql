CREATE TYPE STATUS AS ENUM ('pending', 'sent', 'failed');

CREATE TABLE emails (
  id SERIAL PRIMARY KEY,
  to_address VARCHAR(255) NOT NULL,
  subject VARCHAR(255) NOT NULL,
  body VARCHAR(255) NOT NULL,
  status STATUS NOT NULL DEFAULT 'pending'
);