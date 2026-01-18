-- Rollback: Revert model ID to user-provided format
-- Note: This is a best-effort rollback. The original user-provided IDs
-- are preserved in the slug column.

BEGIN;

-- Drop the unique constraint
DROP INDEX IF EXISTS idx_model_owner_id_unique;

-- Restore id from slug (best effort - original user-provided ID was stored there)
UPDATE model
SET id = slug
WHERE slug IS NOT NULL AND slug != '';

COMMIT;
