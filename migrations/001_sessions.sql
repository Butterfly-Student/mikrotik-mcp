CREATE TABLE IF NOT EXISTS sessions (
    phone      TEXT PRIMARY KEY,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Menyimpan seluruh conversation history per nomor WA
CREATE TABLE IF NOT EXISTS messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    phone        TEXT    NOT NULL,
    role         TEXT    NOT NULL,   -- "user" | "assistant" | "tool"
    content      TEXT    NOT NULL DEFAULT '',
    tool_calls   TEXT,               -- JSON array ToolCall, diisi jika role=assistant
    tool_call_id TEXT,               -- diisi jika role=tool
    name         TEXT,               -- nama tool, diisi jika role=tool
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Audit log setiap tool yang dieksekusi
CREATE TABLE IF NOT EXISTS audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    phone       TEXT    NOT NULL,
    tool_name   TEXT    NOT NULL,
    args        TEXT,
    status      TEXT DEFAULT 'pending',  -- "pending" | "success" | "error"
    error       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_msg_phone_time ON messages(phone, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_phone    ON audit_logs(phone, created_at DESC);
