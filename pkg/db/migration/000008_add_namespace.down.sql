BEGIN;

DROP EXTENSION pg_trgm;
ALTER TABLE model DROP COLUMN namespace_id;
ALTER TABLE model DROP COLUMN namespace_type;

COMMIT;
