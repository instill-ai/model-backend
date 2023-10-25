BEGIN;

ALTER TABLE public.triton_model RENAME TO "inference_model";
ALTER TABLE public.inference_model RENAME CONSTRAINT "fk_triton_model_uid" TO "fk_inference_model_uid";

COMMIT;
