BEGIN;

alter index model_run_model_uid_index rename to model_trigger_model_uid_index;

alter index model_run_requester_uid_index rename to model_trigger_requester_uid_index;

alter index model_run_create_time_index rename to model_trigger_create_time_index;

alter table model_run rename to model_trigger;

comment on column model_trigger.requester_uid is null;

alter table model_trigger drop column runner_uid;

COMMIT;
