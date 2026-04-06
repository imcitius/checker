ALTER TABLE check_definitions
    ADD COLUMN IF NOT EXISTS run_mode TEXT,
    ADD COLUMN IF NOT EXISTS target_regions TEXT;
