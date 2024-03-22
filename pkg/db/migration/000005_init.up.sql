BEGIN;

ALTER TABLE "model"
    ADD COLUMN IF NOT EXISTS "region" VARCHAR(255),
    ADD COLUMN IF NOT EXISTS "hardware_spec" VARCHAR(255),
    ADD COLUMN IF NOT EXISTS "github_link" VARCHAR(1023),
    ADD COLUMN IF NOT EXISTS "link" VARCHAR(1023),
    ADD COLUMN IF NOT EXISTS "license" VARCHAR(255),
    ADD COLUMN IF NOT EXISTS "namespace" VARCHAR(255);
    ADD COLUMN IF NOT EXISTS "version" VARCHAR(255);


CREATE TABLE IF NOT EXISTS "model_version" (
    "uid" UUID PRIMARY KEY,
    "model_uid" UUID NOT NULL,
    "version" INT NOT NULL,
    "digest" VARCHAR(255) NOT NULL,
    "sync_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "create_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "update_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "delete_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
    CONSTRAINT fk_model_uid 
    FOREIGN KEY ("model_uid") 
    REFERENCES model ("uid")
);

CREATE TYPE prediction_state AS ENUM (
    'STATE_SUCCEDED',
    'STATE_CREATED',
    'STATE_RUNNING',
    'STATE_ERROR'
);

CREATE TYPE prediction_source AS ENUM (
    'API',
    'WEB'
);

CREATE TABLE IF NOT EXISTS "prediction" (
    "uid" UUID PRIMARY KEY,
    "model_version_uid" UUID NOT NULL,
    "source" PREDICTION_SOURCE NOT NULL,
    "status" PREDICTION_STATE NOT NULL,
    "run_time" FLOAT NOT NULL,
    "create_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "update_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "delete_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
    "blob_link" VARCHAR(1023),
    "result" VARCHAR(4095),
    CONSTRAINT fk_model_version_uid 
    FOREIGN KEY ("model_version_uid") 
    REFERENCES model_version ("uid")
);

COMMIT;
