BEGIN;

CREATE TYPE valid_statuses AS ENUM ('OFFLINE', 'ONLINE', 'ERROR');

CREATE TABLE IF NOT EXISTS "models" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(256) NOT NULL,
  "optimized" bool DEFAULT false,
  "type" varchar(48),
  "framework" varchar(48),
  "visibility" varchar(48),
  "namespace" varchar(128) NOT NULL
);

COMMENT ON COLUMN "models"."name" IS 'model name label';
COMMENT ON COLUMN "models"."type" IS 'model supported type';
COMMENT ON COLUMN "models"."framework" IS 'model supported framework';
COMMENT ON COLUMN "models"."visibility" IS 'model public or private';

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
