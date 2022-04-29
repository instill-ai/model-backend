BEGIN;

CREATE TYPE valid_status AS ENUM (
  'STATUS_OFFLINE', 
  'STATUS_ONLINE', 
  'STATUS_ERROR'
);
CREATE TYPE valid_visibility AS ENUM (
  'VISIBILITY_PUBLIC', 
  'VISIBILITY_PRIVATE'
);
CREATE TYPE valid_source AS ENUM (
  'SOURCE_GITHUB', 
  'SOURCE_LOCAL'
);

CREATE TABLE IF NOT EXISTS "model" (
  "id" UUID PRIMARY KEY,
  "name" varchar(256) NOT NULL,
  "namespace" varchar(39) NOT NULL,
  "description" varchar(1024),
  "source" VALID_SOURCE DEFAULT 'SOURCE_LOCAL' NOT NULL,
  "visibility" VALID_VISIBILITY DEFAULT 'VISIBILITY_PRIVATE' NOT NULL,
  "config" JSONB,
  "owner" JSONB,
  "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "deleted_at" timestamptz DEFAULT CURRENT_TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS "instance" (
  "id" UUID PRIMARY KEY,
  "model_id" UUID NOT NULL,
  "name" varchar(256) NOT NULL,
  "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "deleted_at" timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
  "status" VALID_STATUS NOT NULL,
  "config" JSONB,
  "task" int NOT NULL,
  UNIQUE ("model_id", "name"),
  CONSTRAINT fk_instance_model_id
    FOREIGN KEY ("model_id")
    REFERENCES model("id")
    ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "triton_model" (
  "id" UUID PRIMARY KEY,
  "name" varchar(256) NOT NULL,
  "version" int NOT NULL,
  "status" VALID_STATUS NOT NULL,
  "model_id" UUID NOT NULL,
  "model_instance" varchar(256) NOT NULL,
  "platform" varchar(256),
  "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "deleted_at" timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
  CONSTRAINT fk_triton_model_instance
    FOREIGN KEY ("model_id", "model_instance")
    REFERENCES instance("model_id", "name")
    ON DELETE CASCADE
);

COMMIT;
