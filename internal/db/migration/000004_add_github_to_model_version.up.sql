BEGIN;

ALTER TABLE "versions" ADD "github" JSONB;

COMMIT;
