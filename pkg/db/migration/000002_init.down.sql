BEGIN;

ALTER TABLE public.inference_model RENAME TO "triton_model";

COMMIT;
