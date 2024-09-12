BEGIN;

alter table model_trigger
add runner_uid uuid;

comment on column model_trigger.requester_uid is 'run by namespace, which is the credit owner';

update model_trigger
set runner_uid = requester_uid
where runner_uid is null;

alter index model_trigger_model_uid_index rename to model_run_model_uid_index;

alter index model_trigger_requester_uid_index rename to model_run_requester_uid_index;

alter index model_trigger_create_time_index rename to model_run_create_time_index;

alter table model_trigger
rename to model_run;

COMMIT;
