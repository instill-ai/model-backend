BEGIN;

CREATE TYPE valid_types AS ENUM ('tensorrt');
CREATE TYPE valid_frameworks AS ENUM ('pytorch', 'tensorflow');
CREATE TYPE valid_visibilities AS ENUM ('public', 'private');
CREATE TYPE valid_statuses AS ENUM ('offline', 'online', 'error');

CREATE TABLE IF NOT EXISTS "models" (
  "id" varchar(64),
  "name" varchar(256) NOT NULL,
  "optimized" bool DEFAULT false NOT NULL,
  "type" VALID_TYPES NOT NULL,
  "framework" VALID_FRAMEWORKS NOT NULL,
  "created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "icon" varchar(512),
  "organization" varchar(256) NOT NULL,
  "description" varchar(1024),
  "visibility" VALID_VISIBILITIES NOT NULL,
  "author" varchar(128)
  PRIMARY KEY ("id", "author")
);

COMMENT ON COLUMN "models"."name" IS 'model name label';
COMMENT ON COLUMN "models"."type" IS 'model supported type';
COMMENT ON COLUMN "models"."framework" IS 'model supported framework';
COMMENT ON COLUMN "models"."icon" IS 'model icon picture url';
COMMENT ON COLUMN "models"."organization" IS 'organization in which model belong to';
COMMENT ON COLUMN "models"."description" IS 'model description';
COMMENT ON COLUMN "models"."visibility" IS 'model public or private';
COMMENT ON COLUMN "models"."author" IS 'model author';

CREATE TABLE IF NOT EXISTS "versions" (
  "model_id" varchar(64) NOT NULL,
  "version" int NOT NULL,
  "description" varchar(1024),
  "created_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" timestamp DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "status" VALID_STATUSES NOT NULL,
  "metadata" JSON,
  PRIMARY KEY ("model_id", "version"),
  CONSTRAINT fk_version_model_id
    FOREIGN KEY (model_id)
    REFERENCES models (id)
    ON DELETE CASCADE
);

COMMENT ON COLUMN "versions"."version" IS 'model version';
COMMENT ON COLUMN "versions"."description" IS 'model version description';
COMMENT ON COLUMN "versions"."status" IS 'model version status';
COMMENT ON COLUMN "versions"."metadata" IS 'model version metadata';

COMMIT;
