BEGIN;

ALTER TYPE "valid_status" RENAME TO "valid_statuses";
ALTER TABLE "model" ALTER COLUMN namespace TYPE varchar(128);
ALTER TABLE "version" ALTER COLUMN "metadata" TYPE JSON;
ALTER TABLE "version" ALTER COLUMN "github" TYPE JSON;

ALTER TABLE "model" DROP "created_at";
ALTER TABLE "model" DROP "updated_at";
ALTER TABLE "model" DROP "deleted_at";
ALTER TABLE "version" DROP "deleted_at";
ALTER TABLE "version" DROP "id";
ALTER TABLE "triton_model" DROP "created_at";
ALTER TABLE "triton_model" DROP "updated_at";
ALTER TABLE "triton_model" DROP "deleted_at";

COMMIT;
