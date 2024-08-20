BEGIN;

DROP TABLE IF EXISTS model_trigger;
DROP TYPE IF EXISTS valid_trigger_status;
DROP TYPE IF EXISTS valid_trigger_source;

ALTER TABLE model DROP COLUMN number_of_runs;
ALTER TABLE model DROP COLUMN last_run_time;
DROP INDEX IF EXISTS model_number_of_runs;
DROP INDEX IF EXISTS model_last_run_time;

COMMIT;
