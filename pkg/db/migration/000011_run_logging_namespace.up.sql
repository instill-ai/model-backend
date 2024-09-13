BEGIN;

alter table model_trigger
add runner_uid uuid;

comment on column model_trigger.requester_uid is 'run by namespace, which is the credit owner';

update model_trigger
set runner_uid = requester_uid
where runner_uid is null;

COMMIT;
