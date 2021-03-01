create table items (
    id int not null,
    processed_at text not null,
    api_version text not null,
    channel text not null,
    item text not null,
    primary key (id, processed_at)
);

create view features as
select
    id,
    channel,
    api_version,
    processed_at,
    json_extract(item, '$.time') as created_at,
    coalesce(json_extract(item, '$.dead'), false) as dead,
    coalesce(json_extract(item, '$.deleted'), false) as deleted,
    json_extract(item, '$.type') as item_type,
    json_extract(item, '$.title') as title,
    json_extract(item, '$.url') as url,
    json_extract(item, '$.text') as text,
    json_extract(item, '$.score') as score,
    case
        when json_extract(item, '$.type') = 'story' then json_extract(item, '$.descendants')
        when json_extract(item, '$.type') = 'comment' then json_array_length(json_extract(item, '$.kids'))
        else null
    end as replies
from items;

create view stories as
select
    id,
    channel,
    processed_at,
    datetime(created_at, 'unixepoch') as created_at,
    title,
    url,
    text,
    score,
    replies
from features
where
    dead = false
    and deleted = false
    and item_type = 'story';
