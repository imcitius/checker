ALTER TABLE check_definitions
    DROP COLUMN IF EXISTS run_mode,
    DROP COLUMN IF EXISTS target_regions;
