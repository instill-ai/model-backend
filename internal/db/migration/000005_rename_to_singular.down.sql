BEGIN;

ALTER TABLE "model" RENAME TO "models";
ALTER TABLE "version" RENAME TO "versions";
ALTER TABLE "triton_model" RENAME TO "t_models";

COMMIT;
