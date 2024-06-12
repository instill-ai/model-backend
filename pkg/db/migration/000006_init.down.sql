BEGIN;

DROP TABLE IF EXISTS tag;
DROP INDEX IF EXISTS tag_model_uid;
DROP INDEX IF EXISTS tag_tag_name;
DROP INDEX IF EXISTS tag_unique_model_tag;
DROP INDEX IF EXISTS version_model_uid;
DROP INDEX IF EXISTS version_unique_model_version;

COMMIT;
