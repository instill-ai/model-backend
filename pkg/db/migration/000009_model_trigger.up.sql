BEGIN;

create type valid_trigger_status as enum ('TRIGGER_STATUS_COMPLETED', 'TRIGGER_STATUS_FAILED', 'TRIGGER_STATUS_PROCESSING', 'TRIGGER_STATUS_QUEUED');
create type valid_trigger_source as enum ('TRIGGER_SOURCE_CONSOLE', 'TRIGGER_SOURCE_API');

create table model_trigger
(
    uid                 uuid primary key         default gen_random_uuid(),
    model_uid           uuid                                               not null,
    model_version       varchar(255)                                       not null,
    status              valid_trigger_status                               not null,
    source              valid_trigger_source                               not null,
    total_duration      bigint,
    end_time            timestamp with time zone,
    requester_uid       uuid                                               not null,
    input_reference_id  varchar(255)                                       not null,
    output_reference_id varchar(255)                                       null,
    error               text                     default ''::text,
    create_time         timestamp with time zone default CURRENT_TIMESTAMP not null,
    update_time         timestamp with time zone default CURRENT_TIMESTAMP not null
);

comment on column model_trigger.total_duration is 'in milliseconds';

create index model_trigger_model_uid_index
    on model_trigger (model_uid);

create index model_trigger_requester_uid_index
    on model_trigger (requester_uid);

create index model_trigger_create_time_index
    on model_trigger (create_time);

COMMIT;
