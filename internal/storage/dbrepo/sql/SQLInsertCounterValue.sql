insert into metrics (id, mtype, delta)
values ($1, 'counter', $2) on conflict (id) do
update
set delta = EXCLUDED.delta + metrics.delta;