insert into metrics (id, mtype, value)
values ($1, 'gauge', $2) on conflict (id) do
update
set value = EXCLUDED.value;