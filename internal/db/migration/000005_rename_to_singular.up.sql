BEGIN;

ALTER TABLE "models" RENAME TO "model";
ALTER TABLE "versions" RENAME TO "version";
ALTER TABLE "t_models" RENAME TO "triton_model";

COMMIT;
