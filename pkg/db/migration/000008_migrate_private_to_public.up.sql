BEGIN;

ALTER TABLE model ALTER COLUMN visibility SET DEFAULT 'VISIBILITY_PUBLIC';
UPDATE model SET visibility = 'VISIBILITY_PUBLIC';

COMMIT;
