-- Docker will execute it once, only if the database volume is empty.
CREATE USER quadmin WITH PASSWORD 'quadmin';
CREATE DATABASE mailqu OWNER quadmin;
GRANT ALL PRIVILEGES ON DATABASE mailqu TO quadmin;