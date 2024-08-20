BEGIN;

ALTER TYPE valid_task ADD VALUE 'TASK_CHAT';
ALTER TYPE valid_task ADD VALUE 'TASK_COMPLETION';
ALTER TYPE valid_task ADD VALUE 'TASK_EMBEDDING';
ALTER TYPE valid_task ADD VALUE 'TASK_CUSTOM';

CREATE TYPE valid_trigger_status AS ENUM ('RUN_STATUS_COMPLETED', 'RUN_STATUS_FAILED', 'RUN_STATUS_PROCESSING', 'RUN_STATUS_QUEUED');
CREATE TYPE valid_trigger_source AS ENUM ('RUN_SOURCE_CONSOLE', 'RUN_SOURCE_API');

CREATE TABLE model_trigger
(
    uid uuid PRIMARY KEY,
    model_uid uuid NOT NULL,
    model_version varchar(255) NOT NULL,
    status valid_trigger_status NOT NULL,
    source valid_trigger_source NOT NULL,
    total_duration bigint,
    end_time timestamp with time zone,
    requester_uid uuid NOT NULL,
    input_reference_id varchar(255) NOT NULL,
    output_reference_id varchar(255) NULL,
    error text DEFAULT ''::text,
    create_time timestamp with time zone DEFAULT current_timestamp NOT NULL,
    update_time timestamp with time zone DEFAULT current_timestamp NOT NULL
);

COMMENT ON COLUMN model_trigger.total_duration IS 'in milliseconds';

CREATE INDEX model_trigger_model_uid_index
ON model_trigger (model_uid);

CREATE INDEX model_trigger_requester_uid_index
ON model_trigger (requester_uid);

CREATE INDEX model_trigger_create_time_index
ON model_trigger (create_time);

ALTER TABLE model ADD COLUMN number_of_runs integer DEFAULT 0;
ALTER TABLE model ADD COLUMN last_run_time timestamptz DEFAULT '0001-01-01T00:00:00Z';
CREATE INDEX model_number_of_runs ON model (number_of_runs);
CREATE INDEX model_last_run_time ON model (last_run_time);

COMMIT;
