BEGIN;

ALTER TABLE model_trigger
ADD runner_uid uuid;

COMMENT ON COLUMN model_trigger.requester_uid IS 'run by namespace, which is the credit owner';

UPDATE model_trigger
SET runner_uid = requester_uid
WHERE runner_uid IS NULL;

COMMIT;
