BEGIN;

comment on column model_trigger.requester_uid is null;

alter table model_trigger drop column runner_uid;

COMMIT;
