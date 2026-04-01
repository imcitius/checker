CREATE TABLE escalation_policies (
  id SERIAL PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  steps TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE escalation_notifications (
  id SERIAL PRIMARY KEY,
  check_uuid TEXT NOT NULL,
  policy_name TEXT NOT NULL,
  step_index INTEGER NOT NULL,
  notified_at TIMESTAMPTZ NOT NULL,
  UNIQUE(check_uuid, policy_name, step_index, notified_at)
);

ALTER TABLE check_definitions ADD COLUMN escalation_policy_name TEXT;
