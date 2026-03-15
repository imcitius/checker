CREATE TABLE escalation_policies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT UNIQUE NOT NULL,
  steps TEXT NOT NULL,
  created_at DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE escalation_notifications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  check_uuid TEXT NOT NULL,
  policy_name TEXT NOT NULL,
  step_index INTEGER NOT NULL,
  notified_at DATETIME NOT NULL,
  UNIQUE(check_uuid, policy_name, step_index, notified_at)
);

ALTER TABLE check_definitions ADD COLUMN escalation_policy_name TEXT;
