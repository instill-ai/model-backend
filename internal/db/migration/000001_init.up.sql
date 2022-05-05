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
  'TASK_DETECTION'
);

CREATE TABLE IF NOT EXISTS "model_definition" (
  "uid" UUID PRIMARY KEY,
  "title" varchar(255) NOT NULL,
  "documentation_url" VARCHAR(1024) NULL,
  "icon" VARCHAR(1024) NULL,
  "spec" JSONB NOT NULL,
  "public" bool,
  "custom" bool,
  "create_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" timestamptz DEFAULT CURRENT_TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS "model" (
  "uid" UUID PRIMARY KEY,
  "id" VARCHAR(63) NOT NULL,
  "description" varchar(1024),
  "model_definition" UUID NOT NULL,
  "configuration" JSONB NOT NULL,
  "visibility" VALID_VISIBILITY DEFAULT 'VISIBILITY_PRIVATE' NOT NULL,
  "owner" VARCHAR(255) NULL,
  "create_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
  UNIQUE ("id", "owner"),
  CONSTRAINT fk_model_definition
    FOREIGN KEY ("model_definition")
    REFERENCES model_definition("uid")
);

CREATE TABLE IF NOT EXISTS "model_instance" (
  "uid" UUID PRIMARY KEY,
  "id" VARCHAR(63) NOT NULL,
  "state" VALID_STATE NOT NULL,
  "task" VALID_TASK NOT NULL,
  "model_definition" varchar(255) NULL,
  "configuration" JSONB NOT NULL,
  "create_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "update_time" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "delete_time" timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
  "model_uid" UUID NOT NULL,
  UNIQUE ("model_uid", "id"),
  CONSTRAINT fk_instance_model_id
    FOREIGN KEY ("model_uid")
    REFERENCES model("uid")
    ON DELETE CASCADE
);

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
  CONSTRAINT fk_triton_model_instance
    FOREIGN KEY ("model_instance_uid")
    REFERENCES model_instance("uid")
    ON DELETE CASCADE
);

COMMIT;
