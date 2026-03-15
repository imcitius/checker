CREATE TABLE escalation_policies (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) UNIQUE NOT NULL,
  steps JSONB NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE escalation_notifications (
  id SERIAL PRIMARY KEY,
  check_uuid VARCHAR(36) NOT NULL,
  policy_name VARCHAR(100) NOT NULL,
  step_index INT NOT NULL,
  notified_at TIMESTAMP NOT NULL,
  UNIQUE(check_uuid, policy_name, step_index, notified_at)
);

ALTER TABLE check_definitions ADD COLUMN escalation_policy_name VARCHAR(100);
