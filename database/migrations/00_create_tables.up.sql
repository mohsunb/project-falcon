create table channels
(
    id                 uuid primary key,
    name               varchar(100) not null,
    position           int          not null,
    creation_timestamp timestamp    not null
);
create table messages
(
    id                 uuid primary key,
    message            text      not null,
    creation_timestamp timestamp not null,
    channel_id         uuid      not null references channels
);
