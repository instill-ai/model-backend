BEGIN;

ALTER TABLE "model" RENAME TO "models";
ALTER TABLE "version" RENAME TO "versions";
ALTER TABLE "triton_model" RENAME TO "t_models";

ALTER TABLE "model" RENAME CONSTRAINT "model_pkey" TO "models_pkey";
ALTER TABLE "version" RENAME CONSTRAINT "version_pkey" TO "versions_pkey";
ALTER TABLE "triton_model" RENAME CONSTRAINT "fk_triton_model_version" TO "fk_tmodel_version";

COMMIT;
