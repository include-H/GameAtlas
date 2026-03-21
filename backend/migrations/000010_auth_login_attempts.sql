CREATE TABLE IF NOT EXISTS auth_login_attempts (
  source_key TEXT PRIMARY KEY,
  fail_count INTEGER NOT NULL DEFAULT 0,
  first_failed_unix INTEGER NOT NULL DEFAULT 0,
  last_failed_unix INTEGER NOT NULL DEFAULT 0,
  locked_until_unix INTEGER NOT NULL DEFAULT 0,
  expires_at_unix INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_login_attempts_expires
ON auth_login_attempts (expires_at_unix);
