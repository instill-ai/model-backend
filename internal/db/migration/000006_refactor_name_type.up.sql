BEGIN;

CREATE TYPE visibility AS ENUM ('VISIBILITY_PUBLIC', 'VISIBILITY_PRIVATE');
CREATE TYPE source AS ENUM ('SOURCE_GITHUB', 'SOURCE_LOCAL');

ALTER TYPE "valid_statuses" RENAME TO "valid_status";
ALTER TABLE "model" ALTER COLUMN "namespace" TYPE varchar(39);
ALTER TABLE "version" ALTER COLUMN "metadata" TYPE JSONB;
ALTER TABLE "version" ALTER COLUMN "github" TYPE JSONB;

ALTER TABLE "model" ADD "created_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "model" ADD "updated_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "model" ADD "deleted_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "model" ADD "visibility" visibility DEFAULT 'VISIBILITY_PRIVATE' NOT NULL;
ALTER TABLE "model" ADD "source" source DEFAULT 'SOURCE_LOCAL' NOT NULL;
ALTER TABLE "version" ADD "deleted_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "version" ADD "id" SERIAL;
ALTER TABLE "triton_model" ADD "created_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "triton_model" ADD "updated_at" timestamp DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "triton_model" ADD "deleted_at" timestamp DEFAULT CURRENT_TIMESTAMP;

COMMIT;
