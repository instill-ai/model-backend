BEGIN;

DELETE FROM model_version WHERE delete_time IS NOT NULL;

ALTER TABLE model_prediction DROP COLUMN IF EXISTS delete_time;
ALTER TABLE model_prediction DROP COLUMN IF EXISTS model_version_uid;
ALTER TABLE model_prediction ADD COLUMN model_version VARCHAR(255) NOT NULL;
ALTER TABLE model_prediction DROP CONSTRAINT IF EXISTS fk_model_version;


ALTER TABLE model_version DROP COLUMN IF EXISTS delete_time;
ALTER TABLE model_version DROP COLUMN IF EXISTS uid;

CREATE INDEX IF NOT EXISTS version_model_uid ON model_version (model_uid);
CREATE UNIQUE INDEX IF NOT EXISTS version_unique_model_version ON model_version (model_uid, version);

CREATE TABLE IF NOT EXISTS model_tag (
    model_uid UUID NOT NULL,
    tag_name VARCHAR(255) NOT NULL,
    create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);


CREATE INDEX IF NOT EXISTS tag_model_uid ON model_tag (model_uid);
CREATE INDEX IF NOT EXISTS tag_tag_name ON model_tag (tag_name);
CREATE UNIQUE INDEX IF NOT EXISTS tag_unique_model_tag ON model_tag (model_uid, tag_name);

COMMIT;
