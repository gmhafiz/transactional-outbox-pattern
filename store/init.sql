create type mail_status AS enum ('In Queue', 'Started', 'Not Sent', 'Failed', 'Success with Errors', 'Success');
create table mail_deliveries
(
    id         bigint generated always as identity
        primary key,

    start_time timestamptz                    null,
    end_time   timestamptz                    null,
    content    jsonb                          null,
    errors     jsonb                          null,
    status     mail_status default 'In Queue' not null,
    created_at timestamptz                    null,
    updated_at timestamptz                    null
);

create table outbox
(
    id      uuid not null primary key,
    type    text,
    payload jsonb
);
