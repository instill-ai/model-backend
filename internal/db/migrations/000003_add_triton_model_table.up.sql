BEGIN;

CREATE TABLE IF NOT EXISTS "t_models" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(256) NOT NULL,
  "version" int NOT NULL,
  "status" VALID_STATUSES NOT NULL,
  "model_id" int NOT NULL,
  "model_version" int NOT NULL,
  "platform" varchar(256),
  CONSTRAINT fk_tmodel_version
    FOREIGN KEY ("model_id", "model_version")
    REFERENCES versions("model_id", "version")
    ON DELETE CASCADE
);

COMMENT ON COLUMN "t_models"."name" IS 'triton model name';
COMMENT ON COLUMN "t_models"."version" IS 'triton model version';
COMMENT ON COLUMN "t_models"."status" IS 'triton model version status';
COMMENT ON COLUMN "t_models"."platform" IS 'triton model platform';

COMMIT;
