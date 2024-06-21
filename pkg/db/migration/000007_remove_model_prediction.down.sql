BEGIN;

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
