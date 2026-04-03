CREATE TABLE IF NOT EXISTS auth_sessions (
    token TEXT PRIMARY KEY,
    expires_at_unix INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires ON auth_sessions (expires_at_unix);
