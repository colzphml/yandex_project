SELECT id,
    mtype,
    value,
    delta
FROM public.metrics
where id = $1;