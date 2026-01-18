-- Rollback migration: Remove slug and aliases columns

BEGIN;

DROP INDEX IF EXISTS idx_model_slug;

ALTER TABLE model
DROP COLUMN IF EXISTS aliases;

ALTER TABLE model
DROP COLUMN IF EXISTS slug;

COMMIT;
