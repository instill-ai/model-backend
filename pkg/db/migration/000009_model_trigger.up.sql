BEGIN;

create type valid_trigger_status as enum ('TRIGGER_STATUS_COMPLETED', 'TRIGGER_STATUS_FAILED', 'TRIGGER_STATUS_PROCESSING', 'TRIGGER_STATUS_QUEUED');
create type valid_trigger_source as enum ('TRIGGER_SOURCE_CONSOLE', 'TRIGGER_SOURCE_API');

create table model_trigger
(
    uid                 uuid primary key,
    model_uid           uuid                                                                    not null,
    trigger_uid         uuid                                                                    not null,
    model_version       varchar(255)                                                            not null,
    model_task          valid_task                                                              not null,
    model_tags          jsonb                                                                   not null,
    status              valid_trigger_status                                                    not null,
    visibility          valid_visibility         default 'VISIBILITY_PRIVATE'::valid_visibility not null,
    source              valid_trigger_source                                                    not null,
    start_time          timestamp with time zone default CURRENT_TIMESTAMP                      not null,
    total_duration      bigint,
    end_time            timestamp with time zone,
    requester_uid       uuid                                                                    not null,
    input_reference_id  varchar(255)                                                            not null,
    output_reference_id varchar(255)                                                            null,
    credits             numeric                  default 0,
    error               text                     default ''::text,
    create_time         timestamp with time zone default CURRENT_TIMESTAMP                      not null,
    update_time         timestamp with time zone default CURRENT_TIMESTAMP                      not null
);

comment on column model_trigger.total_duration is 'in milliseconds';

create index model_trigger_model_uid_index
    on model_trigger (model_uid);

create index model_trigger_requester_uid_index
    on model_trigger (requester_uid);

create index model_trigger_start_time_index
    on model_trigger (start_time);

COMMIT;
