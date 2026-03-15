CREATE TABLE alert_channels (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT UNIQUE NOT NULL,
  type TEXT NOT NULL,
  config TEXT NOT NULL DEFAULT '{}',
  created_at DATETIME DEFAULT (datetime('now')),
  updated_at DATETIME DEFAULT (datetime('now'))
);
