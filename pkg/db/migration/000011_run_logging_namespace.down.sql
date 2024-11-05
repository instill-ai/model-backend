BEGIN;

COMMENT ON COLUMN model_trigger.requester_uid IS NULL;

ALTER TABLE model_trigger DROP COLUMN runner_uid;

COMMIT;
