BEGIN;

ALTER TABLE "models" RENAME TO "model";
ALTER TABLE "versions" RENAME TO "version";
ALTER TABLE "t_models" RENAME TO "triton_model";

ALTER TABLE "model" RENAME CONSTRAINT "models_pkey" TO "model_pkey";
ALTER TABLE "version" RENAME CONSTRAINT "versions_pkey" TO "version_pkey";
ALTER TABLE "triton_model" RENAME CONSTRAINT "fk_tmodel_version" TO "fk_triton_model_version";

COMMIT;
