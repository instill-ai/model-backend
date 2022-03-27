BEGIN;

CREATE TYPE valid_statuses AS ENUM ('STATUS_OFFLINE', 'STATUS_ONLINE', 'STATUS_ERROR');

CREATE TABLE IF NOT EXISTS "models" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(256) NOT NULL,
  "namespace" varchar(128) NOT NULL
);

COMMENT ON COLUMN "models"."name" IS 'model name';
COMMENT ON COLUMN "models"."namespace" IS 'namespace in which model belong to';

CREATE TABLE IF NOT EXISTS "versions" (
  "model_id" int NOT NULL,
  "version" int NOT NULL,
  "description" varchar(1024),
  "created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "status" VALID_STATUSES NOT NULL,
  "metadata" JSON,
  PRIMARY KEY ("model_id", "version"),
  CONSTRAINT fk_version_model_id
    FOREIGN KEY ("model_id")
    REFERENCES models("id")
    ON DELETE CASCADE
);

COMMENT ON COLUMN "versions"."version" IS 'model version';
COMMENT ON COLUMN "versions"."description" IS 'model version description';
COMMENT ON COLUMN "versions"."status" IS 'model version status';
COMMENT ON COLUMN "versions"."metadata" IS 'model version metadata';

COMMIT;
