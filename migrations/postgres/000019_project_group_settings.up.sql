-- Project-level settings (hierarchy: system defaults → project → group → check)
CREATE TABLE IF NOT EXISTS project_settings (
    project TEXT NOT NULL,
    enabled BOOLEAN,                 -- NULL = inherit from system
    duration TEXT,                   -- NULL = inherit
    re_alert_interval TEXT,          -- NULL = inherit
    maintenance_until TIMESTAMPTZ,   -- NULL = not in maintenance
    maintenance_reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project)
);

-- Group-level settings (hierarchy: system defaults → project → group → check)
CREATE TABLE IF NOT EXISTS group_settings (
    project TEXT NOT NULL,
    group_name TEXT NOT NULL,
    enabled BOOLEAN,                 -- NULL = inherit from project/system
    duration TEXT,                   -- NULL = inherit
    re_alert_interval TEXT,          -- NULL = inherit
    maintenance_until TIMESTAMPTZ,   -- NULL = not in maintenance
    maintenance_reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project, group_name)
);
