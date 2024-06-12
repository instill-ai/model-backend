BEGIN;

CREATE TABLE IF NOT EXISTS model_tag (
    model_uid UUID NOT NULL,
    tag_name VARCHAR(255) NOT NULL,
    create_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    update_time TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);


CREATE INDEX tag_model_uid ON model_tag (model_uid);
CREATE INDEX tag_tag_name ON model_tag (tag_name);
CREATE UNIQUE INDEX tag_unique_model_tag ON model_tag (model_uid, tag_name);

CREATE INDEX version_model_uid ON model_version (model_uid);
CREATE UNIQUE INDEX version_unique_model_version ON model_version (model_uid, version);

COMMIT;
