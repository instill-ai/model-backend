BEGIN;

CREATE TYPE valid_state AS ENUM (
  'STATE_OFFLINE', 
  'STATE_ONLINE', 
  'STATE_ERROR'
);
CREATE TYPE valid_visibility AS ENUM (
  'VISIBILITY_PUBLIC', 
  'VISIBILITY_PRIVATE'
);

CREATE TYPE valid_task AS ENUM (
  'TASK_UNSPECIFIED', 
  'TASK_CLASSIFICATION',
  'TASK_DETECTION',
  'TASK_KEYPOINT'
);

CREATE TABLE IF NOT EXISTS "model_definition" (
  "uid" UUID PRIMARY KEY,
  "id" VARCHAR(63) NOT NULL,
  "title" varchar(255) NOT NULL,
  "documentation_url" VARCHAR(1024) NULL,
  "icon" VARCHAR(1024) NULL,
  "model_spec" JSONB NOT NULL,
  "model_instance_spec" JSONB NOT NULL,
  "create_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" timestamptz DEFAULT CURRENT_TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS "model" (
  "uid" UUID PRIMARY KEY,
  "id" VARCHAR(63) NOT NULL,
  "description" varchar(1024),
  "model_definition_uid" UUID NOT NULL,
  "configuration" JSONB NULL,
  "visibility" VALID_VISIBILITY DEFAULT 'VISIBILITY_PRIVATE' NOT NULL,
  "owner" VARCHAR(255) NULL,
  "create_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT fk_model_definition_uid
    FOREIGN KEY ("model_definition_uid")
    REFERENCES model_definition("uid")
);
CREATE UNIQUE INDEX unique_owner_id_delete_time ON model ("owner", "id")
WHERE "delete_time" IS NULL;
CREATE INDEX model_id_create_time_pagination ON model ("id", "create_time");

CREATE TABLE IF NOT EXISTS "model_instance" (
  "uid" UUID PRIMARY KEY,
  "id" VARCHAR(63) NOT NULL,
  "state" VALID_STATE NOT NULL,
  "task" VALID_TASK NOT NULL,
  "configuration" JSONB NULL,
  "create_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
  "model_uid" UUID NOT NULL,
  UNIQUE ("model_uid", "id"),
  CONSTRAINT fk_instance_model_uid
    FOREIGN KEY ("model_uid")
    REFERENCES model("uid")
    ON DELETE CASCADE
);
CREATE INDEX model_instance_id_create_time_pagination ON model_instance ("id", "create_time");

CREATE TABLE IF NOT EXISTS "triton_model" (
  "uid" UUID PRIMARY KEY,
  "name" varchar(255) NOT NULL,
  "version" int NOT NULL,
  "state" VALID_STATE NOT NULL,
  "platform" varchar(256),
  "create_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
  "model_instance_uid" UUID NOT NULL,
  CONSTRAINT fk_triton_model_instance_uid
    FOREIGN KEY ("model_instance_uid")
    REFERENCES model_instance("uid")
    ON DELETE CASCADE
);

COMMIT;
