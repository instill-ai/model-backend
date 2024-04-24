BEGIN;

ALTER TABLE "model"
ADD COLUMN IF NOT EXISTS "region" VARCHAR(255),
ADD COLUMN IF NOT EXISTS "hardware" VARCHAR(255),
ADD COLUMN IF NOT EXISTS "source_url" VARCHAR(255),
ADD COLUMN IF NOT EXISTS "documentation_url" VARCHAR(255),
ADD COLUMN IF NOT EXISTS "license" VARCHAR(255),
ADD COLUMN "readme" TEXT DEFAULT '',
DROP COLUMN "state";

CREATE TABLE IF NOT EXISTS "model_version" (
    "uid" UUID PRIMARY KEY,
    "name" VARCHAR(255) NOT NULL,
    "version" VARCHAR(255) NOT NULL,
    "digest" VARCHAR(255) NOT NULL,
    "create_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "update_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "delete_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NULL,
    "model_uid" UUID NOT NULL,
    CONSTRAINT fk_model_uid
    FOREIGN KEY ("model_uid")
    REFERENCES model ("uid")
    ON DELETE CASCADE
);

CREATE TYPE valid_user_type AS ENUM (
    'OWNER_TYPE_USER',
    'OWNER_TYPE_ORGANIZATION'
);

CREATE TYPE valid_mode AS ENUM (
    'MODE_SYNC',
    'MODE_ASYNC'
);

CREATE TYPE valid_status AS ENUM (
    'STATUS_COMPLETED',
    'STATUS_ERRORED'
);

CREATE TABLE IF NOT EXISTS "model_prediction" (
    "uid" UUID PRIMARY KEY,
    "owner_uid" UUID NOT NULL,
    "owner_type" VALID_USER_TYPE NOT NULL,
    "user_uid" UUID NOT NULL,
    "user_type" VALID_USER_TYPE NOT NULL,
    "mode" VALID_MODE NOT NULL,
    "model_definition_uid" UUID NOT NULL,
    "trigger_time" TIMESTAMPTZ NOT NULL,
    "compute_time_duration" FLOAT(24) NOT NULL,
    "model_task" VALID_TASK NOT NULL,
    "status" VALID_STATUS NOT NULL,
    "input" JSONB NOT NULL,
    "output" JSONB NULL,
    "create_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "update_time" TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "model_uid" UUID NOT NULL,
    "model_version_uid" UUID NOT NULL,
    CONSTRAINT fk_model_version
    FOREIGN KEY ("model_version_uid")
    REFERENCES model_version ("uid")
    ON DELETE CASCADE
);

COMMIT;
